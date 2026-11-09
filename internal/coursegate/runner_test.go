package coursegate

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMatchRejectsMissingAndWrongObservations(t *testing.T) {
	w := map[string]any{"version": float64(2), "committed": true}
	for _, g := range []map[string]any{{}, {"version": float64(3), "committed": true}, {"version": float64(2), "committed": false}, {"version": "2", "committed": true}} {
		if Match(w, g) == nil {
			t.Fatal("accepted wrong observation", g)
		}
	}
	if Match(w, map[string]any{"version": float64(2), "committed": true, "extra": 0}) != nil {
		t.Fatal("subset rejected")
	}
	if Match([]any{float64(1)}, []any{float64(1), float64(2)}) == nil {
		t.Fatal("array length ignored")
	}
}
func TestRunnerSeparatesExpectedFromDriver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		_ = json.NewDecoder(r.Body).Decode(&v)
		if _, ok := v["want"]; ok {
			t.Error("oracle leaked")
		}
		if r.Header.Get("Authorization") != "Bearer test" {
			t.Error("missing auth")
		}
		_, _ = w.Write([]byte(`{"version":2}`))
	}))
	defer srv.Close()
	r, err := New(srv.URL, "test")
	if err != nil {
		t.Fatal(err)
	}
	s := Suite{Lab: "test", Cases: []Case{{Name: "parallel", Steps: []Step{{Parallel: []Step{{Op: "read", Want: map[string]any{"version": float64(2)}}, {Op: "read", Want: map[string]any{"version": float64(2)}}}}}}}}
	var out bytes.Buffer
	if err = r.Run(s, &out); err != nil {
		t.Fatal(err)
	}
	s.Cases[0].Steps = []Step{{Op: "read", Want: map[string]any{"version": float64(9)}}}
	if r.Run(s, &out) == nil {
		t.Fatal("wrong result passed")
	}
}
func TestUnsafeDriverRejected(t *testing.T) {
	for _, u := range []string{"https://example.com", "http://example.com", "http://localhost:8080", "http://user@127.0.0.1", "http://127.0.0.1/execute", "http://127.0.0.1?secret=x"} {
		if _, e := New(u, "test"); e == nil {
			t.Fatal(u)
		}
	}
	if _, e := New("http://127.0.0.1:8099", ""); e == nil {
		t.Fatal("empty token")
	}
}

func TestBoundsAndTransportFailClosed(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		pass   bool
	}{
		{200, `{"ok":true,"p99":12}`, true},
		{200, `{"ok":true,"p99":120}`, false},
		{200, `{"ok":true}`, false},
		{200, `{"ok":true,"p99":"12"}`, false},
		{200, `not json`, false},
		{500, `{"ok":true,"p99":12}`, false},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		r, err := New(srv.URL, "test")
		if err != nil {
			t.Fatal(err)
		}
		err = r.step(Step{Op: "measure", Want: map[string]any{"ok": true}, Bounds: map[string]Bound{"p99": {Min: 0, Max: 50}}})
		srv.Close()
		if (err == nil) != tc.pass {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
}
func TestCommittedSuitesAreValid(t *testing.T) {
	paths, e := filepath.Glob("../../fixtures/gates/lab*.json")
	if e != nil {
		t.Fatal(e)
	}
	if len(paths) != 13 {
		t.Fatalf("want 13 driver suites, got %d", len(paths))
	}
	for _, p := range paths {
		if _, e := Load(p); e != nil {
			t.Fatalf("%s: %v", p, e)
		}
	}
}
func TestInvalidSuiteRejected(t *testing.T) {
	p := filepath.Join(t.TempDir(), "suite.json")
	for _, s := range []string{`{}`, `{"lab":"02","cases":[{"name":"a","steps":[{"op":"read"}]}]}`, `{"lab":"02","cases":[],"typo":true}`} {
		if e := os.WriteFile(p, []byte(s), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := Load(p); e == nil {
			t.Fatal("invalid suite passed")
		}
	}
}
