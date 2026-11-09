//go:build reference

package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/JiangZhan-s/ArenaLab/db/migrations"
	"github.com/JiangZhan-s/ArenaLab/internal/rules"
	_ "github.com/go-sql-driver/mysql"
)

type Store struct {
	DelayMS atomic.Int64
	DB      *sql.DB
	Node    string
	Hook    func(string)
}
type Match struct {
	ID                    string
	Players               [2]int
	State                 rules.State
	Status, Owner, Policy string
	Epoch                 int
	Deadline, Lease       time.Time
}
type Command struct {
	MatchID string `json:"match_id"`
	ID      string `json:"command_id"`
	Version int    `json:"expected_version"`
	Kind    string `json:"kind"`
	DX      int    `json:"dx"`
	DY      int    `json:"dy"`
}
type Reply struct {
	Error     string      `json:"error"`
	Version   int         `json:"version"`
	State     rules.State `json:"state"`
	Committed bool        `json:"committed"`
	Status    string      `json:"status"`
}

func Open(dsn, node string) (*Store, error) {
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(32)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		return nil, e
	}
	return &Store{DB: db, Node: node}, nil
}
func (s *Store) Migrate(ctx context.Context) error {
	for _, q := range strings.Split(migrations.Schema, ";") {
		if strings.TrimSpace(q) != "" {
			if _, e := s.DB.ExecContext(ctx, q); e != nil {
				return e
			}
		}
	}
	var column int
	if e := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='players' AND column_name='ws_consumed'").Scan(&column); e != nil {
		return e
	}
	if column == 0 {
		_, e := s.DB.ExecContext(ctx, "ALTER TABLE players ADD COLUMN ws_consumed BOOLEAN NOT NULL DEFAULT FALSE")
		return e
	}
	return nil
}
func (s *Store) EnsurePlayer(ctx context.Context, id int) error {
	_, e := s.DB.ExecContext(ctx, "INSERT IGNORE INTO players(id) VALUES(?)", id)
	return e
}
func (s *Store) Login(ctx context.Context, id int) (int, error) {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	var epoch int
	if e = tx.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=? FOR UPDATE", id).Scan(&epoch); e != nil {
		return 0, e
	}
	epoch++
	if _, e = tx.ExecContext(ctx, "UPDATE players SET epoch=?,ws_consumed=FALSE WHERE id=?", epoch, id); e != nil {
		return 0, e
	}
	return epoch, tx.Commit()
}
func (s *Store) Create(ctx context.Context, id string, p0, p1 int, policy string) error {
	b, _ := json.Marshal(rules.Initial())
	_, e := s.DB.ExecContext(ctx, "INSERT INTO matches(id,p0,p1,state,deadline,owner,lease_until,policy) VALUES(?,?,?,?,TIMESTAMPADD(SECOND,15,UTC_TIMESTAMP(6)),?,TIMESTAMPADD(SECOND,3,UTC_TIMESTAMP(6)),?)", id, p0, p1, b, s.Node, policy)
	return e
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Match, error) {
	var m Match
	var b []byte
	e := row.Scan(&m.ID, &m.Players[0], &m.Players[1], &b, &m.Status, &m.Deadline, &m.Owner, &m.Epoch, &m.Lease, &m.Policy)
	if e == nil {
		e = json.Unmarshal(b, &m.State)
	}
	return m, e
}

const fields = "id,p0,p1,state,status,deadline,owner,owner_epoch,lease_until,policy"

func (s *Store) Get(ctx context.Context, id string) (Match, error) {
	return scan(s.DB.QueryRowContext(ctx, "SELECT "+fields+" FROM matches WHERE id=?", id))
}
func (s *Store) Action(ctx context.Context, player, epoch, ownerEpoch int, c Command) (Reply, error) {
	fail := func(code string) (Reply, error) { return Reply{Error: code}, nil }
	if c.ID == "" || len(c.ID) > 128 {
		return fail("invalid_command")
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return Reply{}, e
	}
	defer tx.Rollback()
	var current int
	if e = tx.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=? FOR UPDATE", player).Scan(&current); e != nil {
		return Reply{}, e
	}
	if current != epoch {
		return fail("stale_session")
	}
	m, e := scan(tx.QueryRowContext(ctx, "SELECT "+fields+" FROM matches WHERE id=? FOR UPDATE", c.MatchID))
	if errors.Is(e, sql.ErrNoRows) {
		return fail("not_found")
	}
	if e != nil {
		return Reply{}, e
	}
	slot := -1
	for i, p := range m.Players {
		if p == player {
			slot = i
		}
	}
	if slot < 0 {
		return fail("forbidden")
	}
	payload, _ := json.Marshal(c)
	h := sha256.Sum256(payload)
	hash := hex.EncodeToString(h[:])
	var oldHash string
	var old []byte
	e = tx.QueryRowContext(ctx, "SELECT payload_hash,response FROM commands WHERE match_id=? AND player_id=? AND command_id=?", c.MatchID, player, c.ID).Scan(&oldHash, &old)
	if e == nil {
		if hash != oldHash {
			return fail("idempotency_conflict")
		}
		var r Reply
		e = json.Unmarshal(old, &r)
		return r, e
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return Reply{}, e
	}
	var now time.Time
	if e = tx.QueryRowContext(ctx, "SELECT UTC_TIMESTAMP(6)").Scan(&now); e != nil {
		return Reply{}, e
	}
	if m.Owner != s.Node || m.Epoch != ownerEpoch || !now.Before(m.Lease) {
		return fail("stale_owner")
	}
	if m.Status != "running" {
		return fail("finished")
	}
	if c.Version != m.State.Version {
		return fail("stale_version")
	}
	if !now.Before(m.Deadline) {
		return fail("deadline_expired")
	}
	next, e := rules.Apply(m.State, rules.Action{Player: slot, Kind: c.Kind, DX: c.DX, DY: c.DY})
	if e != nil {
		return fail(e.Error())
	}
	status := "running"
	if next.Finished {
		status = "finished"
	}
	r := Reply{Version: next.Version, State: next, Committed: true, Status: status}
	b, _ := json.Marshal(next)
	out, _ := json.Marshal(r)
	if _, e = tx.ExecContext(ctx, "UPDATE matches SET state=?,status=?,deadline=TIMESTAMPADD(SECOND,15,UTC_TIMESTAMP(6)) WHERE id=?", b, status, m.ID); e != nil {
		return Reply{}, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO events(match_id,version,state,reason) VALUES(?,?,?,?)", m.ID, next.Version, b, c.Kind); e != nil {
		return Reply{}, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO commands(match_id,player_id,command_id,payload_hash,response) VALUES(?,?,?,?,?)", m.ID, player, c.ID, hash, out); e != nil {
		return Reply{}, e
	}
	if next.Finished {
		if _, e = tx.ExecContext(ctx, "INSERT INTO results(match_id,result_id,winner) VALUES(?,?,?)", m.ID, "result-"+m.ID, next.Winner); e != nil {
			return Reply{}, e
		}
	}
	if s.Hook != nil {
		s.Hook("before_commit")
	}
	if e = tx.Commit(); e != nil {
		return Reply{Error: "outcome_unknown"}, e
	}
	if s.Hook != nil {
		s.Hook("after_commit_before_ack")
	}
	return r, nil
}
func (s *Store) Lookup(ctx context.Context, player, epoch int, match, id string) (Reply, error) {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return Reply{}, e
	}
	defer tx.Rollback()
	var current int
	if e = tx.QueryRowContext(ctx, "SELECT epoch FROM players WHERE id=? FOR UPDATE", player).Scan(&current); e != nil {
		return Reply{}, e
	}
	if current != epoch {
		return Reply{Error: "stale_session"}, nil
	}
	var p0, p1 int
	if e = tx.QueryRowContext(ctx, "SELECT p0,p1 FROM matches WHERE id=?", match).Scan(&p0, &p1); e != nil {
		return Reply{}, e
	}
	if player != p0 && player != p1 {
		return Reply{Error: "forbidden"}, nil
	}
	var b []byte
	e = tx.QueryRowContext(ctx, "SELECT response FROM commands WHERE match_id=? AND player_id=? AND command_id=?", match, player, id).Scan(&b)
	if errors.Is(e, sql.ErrNoRows) {
		return Reply{Error: "not_found"}, nil
	}
	if e != nil {
		return Reply{}, e
	}
	var r Reply
	e = json.Unmarshal(b, &r)
	return r, e
}
func (s *Store) Settle(ctx context.Context, match string, winner *int) error {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var actual int
	var state string
	if e = tx.QueryRowContext(ctx, "SELECT winner,state FROM results WHERE match_id=? FOR UPDATE", match).Scan(&actual, &state); e != nil {
		return e
	}
	if winner != nil && *winner != actual {
		return fmt.Errorf("result_conflict")
	}
	if state == "applied" {
		return nil
	}
	m, e := scan(tx.QueryRowContext(ctx, "SELECT "+fields+" FROM matches WHERE id=?", match))
	if e != nil {
		return e
	}
	if m.Status != "finished" {
		return fmt.Errorf("not_finished")
	}
	// Players are locked in stable ID order, independent of the winner.
	order := []int{0, 1}
	if m.Players[0] > m.Players[1] {
		order = []int{1, 0}
	}
	for _, slot := range order {
		p := m.Players[slot]
		delta := 0
		if actual >= 0 {
			delta = -10
			if actual == slot {
				delta = 10
			}
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO ledger(match_id,player_id,delta) VALUES(?,?,?)", match, p, delta); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO ratings(player_id,score,version) VALUES(?,?,1) ON DUPLICATE KEY UPDATE score=score+VALUES(score),version=version+1", p, delta); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, "UPDATE results SET state='applied' WHERE match_id=?", match); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM occupancy WHERE match_id=?", match); e != nil {
		return e
	}
	return tx.Commit()
}
