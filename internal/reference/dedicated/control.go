//go:build reference

package dedicated

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"github.com/JiangZhan-s/ArenaLab/internal/rules"
	"strconv"
	"strings"
	"time"
)

type Allocation struct {
	ID         string `json:"allocation_id"`
	Generation int    `json:"generation"`
	Instance   string `json:"instance_id"`
	Match      string `json:"match_id"`
	Endpoint   string `json:"endpoint"`
	Build      string `json:"build"`
	Region     string `json:"region"`
	State      string `json:"state"`
	Error      string `json:"error"`
}
type Ticket struct {
	Player     int    `json:"player_id"`
	Match      string `json:"match_id"`
	Allocation string `json:"allocation_id"`
	Generation int    `json:"generation"`
	Build      string `json:"build"`
	Audience   string `json:"audience"`
	Expires    int64  `json:"exp"`
	JTI        string `json:"jti"`
}

func Key(secret, purpose, instance string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(purpose + ":" + instance))
	return hex.EncodeToString(m.Sum(nil))
}
func Sign(key string, t Ticket) string {
	b, _ := json.Marshal(t)
	payload := base64.RawURLEncoding.EncodeToString(b)
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
func Verify(key, raw string) (Ticket, error) {
	var t Ticket
	p := strings.Split(raw, ".")
	if len(p) != 2 {
		return t, fmt.Errorf("invalid_ticket")
	}
	sig, e := base64.RawURLEncoding.DecodeString(p[1])
	if e != nil {
		return t, fmt.Errorf("invalid_ticket")
	}
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(p[0]))
	if !hmac.Equal(sig, m.Sum(nil)) {
		return t, fmt.Errorf("invalid_ticket")
	}
	b, e := base64.RawURLEncoding.DecodeString(p[0])
	if e != nil {
		return t, e
	}
	if e = json.Unmarshal(b, &t); e != nil {
		return t, e
	}
	if t.Expires <= time.Now().Unix() {
		return t, fmt.Errorf("ticket_expired")
	}
	return t, nil
}

type Control struct {
	Store  *store.Store
	Secret string
}

func (c *Control) Register(ctx context.Context, id, endpoint, build, region string) error {
	if id == "" || len(id) > 64 || build == "" || region == "" {
		return fmt.Errorf("invalid_registration")
	}
	// A never-allocated instance may have been swept unhealthy while the
	// backend restarted. Registration can revive only that idle identity;
	// a consumed/active one-shot instance must never silently be recycled.
	_, e := c.Store.DB.ExecContext(ctx, "INSERT INTO ds_instances(id,endpoint,build,region,status,last_seen) VALUES(?,?,?,?,'ready',UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE status=IF(status='unhealthy' AND NOT EXISTS(SELECT 1 FROM allocations WHERE instance_id=?),'ready',status),last_seen=UTC_TIMESTAMP(6)", id, endpoint, build, region, id)
	return e
}
func (c *Control) Heartbeat(ctx context.Context, id string) error {
	_, e := c.Store.DB.ExecContext(ctx, "UPDATE ds_instances SET last_seen=UTC_TIMESTAMP(6) WHERE id=?", id)
	return e
}
func allocationScan(row interface{ Scan(...any) error }) (Allocation, error) {
	var a Allocation
	var id int
	e := row.Scan(&id, &a.Generation, &a.Instance, &a.Match, &a.Endpoint, &a.Build, &a.Region, &a.State)
	a.ID = "a" + strconv.Itoa(id)
	return a, e
}

const allocationFields = "a.id,a.generation,a.instance_id,a.match_id,d.endpoint,a.build,a.region,a.status"

func (c *Control) Find(ctx context.Context, id string) (Allocation, error) {
	return allocationScan(c.Store.DB.QueryRowContext(ctx, "SELECT "+allocationFields+" FROM allocations a JOIN ds_instances d ON a.instance_id=d.id WHERE a.id=?", strings.TrimPrefix(id, "a")))
}
func (c *Control) Allocate(ctx context.Context, player, epoch int, request, build, region string) (Allocation, error) {
	fail := func(code string) (Allocation, error) { return Allocation{Error: code}, nil }
	if request == "" || len(request) > 128 {
		return fail("invalid_request")
	}
	tx, e := c.Store.DB.BeginTx(ctx, nil)
	if e != nil {
		return Allocation{}, e
	}
	defer tx.Rollback()
	var current int
	p0 := player - player%2
	p1 := p0 + 1
	// Serialize both members in stable ID order, including simultaneous
	// allocation requests from opposite members of the same local pair.
	for _, member := range []int{p0, p1} {
		var memberEpoch int
		if e = tx.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=? FOR UPDATE", member).Scan(&memberEpoch); e != nil {
			return Allocation{}, e
		}
		if member == player {
			current = memberEpoch
		}
	}
	if current != epoch {
		return fail("stale_session")
	}
	old, e := allocationScan(tx.QueryRowContext(ctx, "SELECT "+allocationFields+" FROM allocations a JOIN ds_instances d ON a.instance_id=d.id WHERE a.player_id=? AND a.request_id=?", player, request))
	if e == nil {
		if old.Build != build || old.Region != region {
			return fail("idempotency_conflict")
		}
		return old, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return Allocation{}, e
	}
	// Local demo pairs are explicit: 0/1, 2/3, ... . Production parties need invitations.
	var occupied int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM occupancy WHERE player_id IN (?,?)", p0, p1).Scan(&occupied); e != nil {
		return Allocation{}, e
	}
	if occupied > 0 {
		return fail("already_assigned")
	}
	var instance, endpoint string
	e = tx.QueryRowContext(ctx, "SELECT id,endpoint FROM ds_instances WHERE status='ready' AND build=? AND region=? AND last_seen>TIMESTAMPADD(SECOND,-3,UTC_TIMESTAMP(6)) ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED", build, region).Scan(&instance, &endpoint)
	if errors.Is(e, sql.ErrNoRows) {
		return fail("no_capacity")
	}
	if e != nil {
		return Allocation{}, e
	}
	var random [12]byte
	if _, e = rand.Read(random[:]); e != nil {
		return Allocation{}, e
	}
	match := "ds-" + hex.EncodeToString(random[:])
	b, _ := json.Marshal(rules.Initial())
	if _, e = tx.ExecContext(ctx, "INSERT INTO matches(id,p0,p1,state,status,deadline,owner,lease_until,policy,rules_version) VALUES(?,?,?,?,'running',TIMESTAMPADD(SECOND,60,UTC_TIMESTAMP(6)),?,TIMESTAMPADD(SECOND,3,UTC_TIMESTAMP(6)),'dedicated','realtime-v1')", match, p0, p1, b, instance); e != nil {
		return Allocation{}, e
	}
	result, e := tx.ExecContext(ctx, "INSERT INTO allocations(player_id,request_id,instance_id,generation,match_id,status,build,region) VALUES(?,?,?,1,?,'allocated',?,?)", player, request, instance, match, build, region)
	if e != nil {
		return Allocation{}, e
	}
	id, e := result.LastInsertId()
	if e != nil {
		return Allocation{}, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO occupancy(player_id,match_id) VALUES(?,?),(?,?)", p0, match, p1, match); e != nil {
		return Allocation{}, e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE ds_instances SET status='allocated' WHERE id=?", instance); e != nil {
		return Allocation{}, e
	}
	if e = tx.Commit(); e != nil {
		return Allocation{Error: "outcome_unknown"}, e
	}
	return Allocation{ID: fmt.Sprintf("a%d", id), Generation: 1, Instance: instance, Match: match, Endpoint: endpoint, Build: build, Region: region, State: "allocated"}, nil
}
func (c *Control) Ticket(ctx context.Context, player int, id string) (string, error) {
	a, e := c.Find(ctx, id)
	if e != nil {
		return "", e
	}
	if a.State != "allocated" {
		return "", fmt.Errorf("stale_allocation")
	}
	m, e := c.Store.Get(ctx, a.Match)
	if e != nil {
		return "", e
	}
	if player != m.Players[0] && player != m.Players[1] {
		return "", fmt.Errorf("forbidden")
	}
	var random [16]byte
	if _, e = rand.Read(random[:]); e != nil {
		return "", e
	}
	return Sign(Key(c.Secret, "ticket", a.Instance), Ticket{player, a.Match, a.ID, a.Generation, a.Build, a.Instance, time.Now().Add(30 * time.Second).Unix(), hex.EncodeToString(random[:])}), nil
}
func (c *Control) Result(ctx context.Context, instance, id string, generation, winner int) error {
	if winner < -1 || winner > 1 {
		return fmt.Errorf("invalid_result")
	}
	tx, e := c.Store.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var own, match, status string
	var gen int
	if e = tx.QueryRowContext(ctx, "SELECT instance_id,match_id,status,generation FROM allocations WHERE id=? FOR UPDATE", strings.TrimPrefix(id, "a")).Scan(&own, &match, &status, &gen); e != nil {
		return e
	}
	if own != instance || gen != generation || status == "aborted" {
		return fmt.Errorf("stale_allocation")
	}
	if status == "finished" {
		var old int
		if e = tx.QueryRowContext(ctx, "SELECT winner FROM results WHERE match_id=?", match).Scan(&old); e != nil {
			return e
		}
		if old != winner {
			return fmt.Errorf("result_conflict")
		}
		return nil
	}
	// Check inside the result transaction: an overdue heartbeat must not win
	// merely because the asynchronous expiry sweeper has not run yet.
	var live bool
	if e = tx.QueryRowContext(ctx, "SELECT last_seen>TIMESTAMPADD(SECOND,-3,UTC_TIMESTAMP(6)) FROM ds_instances WHERE id=? FOR UPDATE", instance).Scan(&live); e != nil {
		return e
	}
	if !live {
		return fmt.Errorf("stale_allocation")
	}
	if _, e = tx.ExecContext(ctx, "UPDATE allocations SET status='finished' WHERE id=?", strings.TrimPrefix(id, "a")); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE matches SET status='finished',state=JSON_SET(state,'$.finished',CAST('true' AS JSON),'$.winner',?) WHERE id=?", winner, match); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO results(match_id,result_id,winner) VALUES(?,?,?)", match, "result-"+match, winner); e != nil {
		return e
	}
	return tx.Commit()
}
func (c *Control) Expire(ctx context.Context) error {
	tx, e := c.Store.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, "SELECT a.id,a.match_id FROM allocations a JOIN ds_instances d ON a.instance_id=d.id WHERE a.status='allocated' AND d.last_seen<=TIMESTAMPADD(SECOND,-3,UTC_TIMESTAMP(6)) FOR UPDATE")
	if e != nil {
		return e
	}
	type item struct {
		id    int
		match string
	}
	var list []item
	for rows.Next() {
		var i item
		if e = rows.Scan(&i.id, &i.match); e != nil {
			rows.Close()
			return e
		}
		list = append(list, i)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, i := range list {
		if _, e = tx.ExecContext(ctx, "UPDATE allocations SET status='aborted' WHERE id=?", i.id); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE matches SET status='aborted' WHERE id=?", i.match); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "DELETE FROM occupancy WHERE match_id=?", i.match); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, "UPDATE ds_instances SET status='unhealthy' WHERE last_seen<=TIMESTAMPADD(SECOND,-3,UTC_TIMESTAMP(6))"); e != nil {
		return e
	}
	return tx.Commit()
}
