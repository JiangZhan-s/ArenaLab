package acceptance

import (
	"encoding/json"
	"github.com/JiangZhan-s/ArenaLab/internal/rules"
	"os"
	"testing"
)

type fixture struct {
	Name   string       `json:"name"`
	Before rules.State  `json:"before"`
	Action rules.Action `json:"action"`
	After  rules.State  `json:"after"`
	Error  string       `json:"error"`
}

func cases(t *testing.T) []fixture {
	t.Helper()
	b, err := os.ReadFile("../../../fixtures/rules.json")
	if err != nil {
		t.Fatal(err)
	}
	var f []fixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	if len(f) < 10 {
		t.Fatal("fixed fixture coverage missing")
	}
	return f
}
func matches(f fixture, s rules.State, err error) bool {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return s == f.After && message == f.Error
}
func TestFixtureIntegrity(t *testing.T) {
	names := map[string]bool{}
	for _, f := range cases(t) {
		if f.Name == "" || names[f.Name] {
			t.Fatal("empty/duplicate fixture name")
		}
		names[f.Name] = true
		if f.Error != "" && f.Before != f.After {
			t.Fatalf("%s: rejection mutates state", f.Name)
		}
		if f.Error == "" && f.After.Version != f.Before.Version+1 {
			t.Fatalf("%s: invalid expected version", f.Name)
		}
	}
}

// These deliberately broken candidates prove the fixtures reject selected defects.
// This is NOT a substitute for business/process mutation coverage in later labs.
func TestFixturesRejectBrokenCandidates(t *testing.T) {
	candidates := map[string]func(rules.State, rules.Action) (rules.State, error){
		"accept_without_transition": func(s rules.State, a rules.Action) (rules.State, error) { return s, nil },
		"always_advance": func(s rules.State, a rules.Action) (rules.State, error) {
			s.Ply++
			s.Version++
			s.Turn = 1 - s.Turn
			return s, nil
		},
	}
	for name, candidate := range candidates {
		rejected := false
		for _, f := range cases(t) {
			s, e := candidate(f.Before, f.Action)
			if !matches(f, s, e) {
				rejected = true
			}
		}
		if !rejected {
			t.Fatalf("fixtures failed to reject %s", name)
		}
	}
}
