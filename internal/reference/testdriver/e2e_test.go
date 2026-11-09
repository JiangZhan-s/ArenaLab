//go:build reference && e2e

package testdriver

import (
	"bytes"
	"github.com/JiangZhan-s/ArenaLab/internal/coursegate"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealProcessGates(t *testing.T) {
	admin := os.Getenv("ARENALAB_ADMIN_DSN")
	if admin == "" {
		t.Fatal("real MySQL ARENALAB_ADMIN_DSN required; no memory fallback")
	}
	root, e := filepath.Abs("../../..")
	if e != nil {
		t.Fatal(e)
	}
	artifacts := os.Getenv("ARENALAB_ARTIFACTS")
	if artifacts == "" {
		artifacts = filepath.Join(root, "artifacts/raw/reference")
	}
	d := New(admin, filepath.Join(root, "bin/arena-server"), "arenalab-local-development", "arenalab-local-test-token", artifacts)
	defer d.Close()
	srv := httptest.NewServer(d.Handler())
	defer srv.Close()
	r, e := coursegate.New(srv.URL, d.Token)
	if e != nil {
		t.Fatal(e)
	}
	ids := os.Getenv("ARENALAB_GATES")
	if ids == "" {
		ids = "02,03,04,05,06,07,08,09,10,11,12,13,14"
	}
	for _, id := range strings.Split(ids, ",") {
		t.Run("lab"+id, func(t *testing.T) {
			suite, e := coursegate.Load(filepath.Join(root, "fixtures/gates/lab"+id+".json"))
			if e != nil {
				t.Fatal(e)
			}
			var out bytes.Buffer
			e = r.Run(suite, &out)
			t.Log(out.String())
			if e != nil {
				t.Fatal(e)
			}
		})
	}
}
