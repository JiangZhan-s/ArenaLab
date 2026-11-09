//go:build reference && e2e

package testdriver

import (
	"context"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/client"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpiryAndConnectionRegressions(t *testing.T) {
	admin := os.Getenv("ARENALAB_ADMIN_DSN")
	if admin == "" {
		t.Fatal("real MySQL required")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	artifacts := os.Getenv("ARENALAB_ARTIFACTS")
	if artifacts == "" {
		artifacts = filepath.Join(root, "artifacts/raw/reference")
	}
	d := New(admin, filepath.Join(root, "bin/arena-server"), "arenalab-local-development", "arenalab-local-test-token", artifacts)
	defer d.Close()
	ctx := context.Background()
	if _, err = d.reset(ctx, map[string]any{"profile": "turn", "seed": "empty"}); err != nil {
		t.Fatal(err)
	}
	// Stop the maintenance process so the test proves the transaction itself,
	// not a race won by a background sweeper.
	d.stop(d.processes["node1"])
	st := &store.Store{DB: d.DB, Node: "node1"}
	for _, p := range []int{0, 1} {
		if _, err = st.Enqueue(ctx, p, 0, "ready-regression"); err != nil {
			t.Fatal(err)
		}
	}
	if err = st.MatchQueued(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = d.DB.Exec("UPDATE matches SET deadline=TIMESTAMPADD(SECOND,-1,UTC_TIMESTAMP(6))"); err != nil {
		t.Fatal(err)
	}
	if err = st.ExpireReady(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = d.DB.QueryRow("SELECT COUNT(*) FROM occupancy").Scan(&count); err != nil || count != 0 {
		t.Fatalf("occupancy=%d err=%v", count, err)
	}
	if err = d.DB.QueryRow("SELECT COUNT(*) FROM matches WHERE status='aborted'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("aborted=%d err=%v", count, err)
	}
	if err = d.DB.QueryRow("SELECT COUNT(*) FROM results").Scan(&count); err != nil || count != 0 {
		t.Fatalf("unexpected result=%d err=%v", count, err)
	}

	c := &dedicated.Control{Store: st, Secret: d.Secret}
	if err = c.Register(ctx, "expiry-ds", "http://127.0.0.1:1", "ds-v1", "local"); err != nil {
		t.Fatal(err)
	}
	a, err := c.Allocate(ctx, 0, 0, "expiry-request", "ds-v1", "local")
	if err != nil || a.Error != "" {
		t.Fatalf("allocation=%+v err=%v", a, err)
	}
	if _, err = d.DB.Exec("UPDATE ds_instances SET last_seen=TIMESTAMPADD(SECOND,-4,UTC_TIMESTAMP(6)) WHERE id='expiry-ds'"); err != nil {
		t.Fatal(err)
	}
	if err = c.Result(ctx, "expiry-ds", a.ID, a.Generation, 0); err == nil || err.Error() != "stale_allocation" {
		t.Fatalf("expired result: %v", err)
	}
	if err = d.DB.QueryRow("SELECT COUNT(*) FROM results").Scan(&count); err != nil || count != 0 {
		t.Fatalf("unexpected result=%d err=%v", count, err)
	}
	if err = c.Expire(ctx); err != nil {
		t.Fatal(err)
	}
	if err = d.DB.QueryRow("SELECT COUNT(*) FROM occupancy").Scan(&count); err != nil || count != 0 {
		t.Fatalf("DS occupancy=%d err=%v", count, err)
	}

	if _, err = d.reset(ctx, map[string]any{"profile": "turn", "seed": "running"}); err != nil {
		t.Fatal(err)
	}
	if _, err = d.connect(ctx, map[string]any{"player": 0}); err != nil {
		t.Fatal(err)
	}
	existing := d.defaults[0]
	duplicate, resp, err := client.Dial("ws"+strings.TrimPrefix(d.processes["node1"].base, "http")+"/v1/ws", existing.Token, "")
	if duplicate != nil {
		duplicate.Close()
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil || resp == nil || resp.StatusCode != 403 {
		t.Fatalf("ticket replay status=%v err=%v", resp, err)
	}
}
