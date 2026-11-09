package main

import (
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/coursegate"
	"os"
	"os/exec"
	"path/filepath"
)

type lab struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}
type manifest struct {
	Labs []lab `json:"labs"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: go run ./cmd/course check | gate NN | gate-all")
	}
	b, err := os.ReadFile("course.json")
	if err != nil {
		return err
	}
	var c manifest
	if err = json.Unmarshal(b, &c); err != nil {
		return err
	}
	if len(c.Labs) != 15 {
		return fmt.Errorf("expected 15 labs")
	}
	if args[0] == "check" {
		seen := map[string]bool{}
		for _, l := range c.Labs {
			if seen[l.ID] {
				return fmt.Errorf("duplicate lab %s", l.ID)
			}
			seen[l.ID] = true
			if l.ID != "00" && l.ID != "01" {
				s, e := coursegate.Load(filepath.Join("fixtures", "gates", "lab"+l.ID+".json"))
				if e != nil {
					return e
				}
				if s.Lab != l.ID {
					return fmt.Errorf("suite lab mismatch")
				}
			}
			for _, name := range []string{"README.md", "tasks.md", "acceptance.json", "review.md", "interview.md", "reading.md"} {
				p := filepath.Join(l.Path, name)
				data, e := os.ReadFile(p)
				if e != nil {
					return e
				}
				if len(data) == 0 {
					return fmt.Errorf("empty %s", p)
				}
				if name == "acceptance.json" {
					var a struct {
						Lab      string `json:"lab"`
						Required string `json:"required_result"`
					}
					if e = json.Unmarshal(data, &a); e != nil {
						return e
					}
					if a.Lab != l.ID || a.Required == "" {
						return fmt.Errorf("invalid acceptance %s", p)
					}
				}
			}
		}
		for _, p := range []string{"contracts/SEMANTICS.md", "contracts/invariants.json", "contracts/websocket.schema.json", "fixtures/rules.json", "PROGRESS.md"} {
			if _, err := os.ReadFile(p); err != nil {
				return err
			}
		}
		fmt.Println("PASS: course structure only; no business gate is implied")
		return nil
	}
	if args[0] == "gate-all" && len(args) == 1 {
		failed := 0
		for _, l := range c.Labs {
			if err := run([]string{"gate", l.ID}); err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "FAIL gate %s: %v\n", l.ID, err)
			}
		}
		if failed != 0 {
			return fmt.Errorf("%d/%d gates failed", failed, len(c.Labs))
		}
		fmt.Println("PASS all executable gates; manual defense and browser/UE evidence remain separate")
		return nil
	}
	if args[0] == "gate" && len(args) == 2 {
		id := args[1]
		if id == "00" {
			return modelGate()
		}
		found := false
		for _, l := range c.Labs {
			if l.ID == id {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("unknown lab %s", id)
		}
		if id != "01" {
			s, err := coursegate.Load(filepath.Join("fixtures", "gates", "lab"+id+".json"))
			if err != nil {
				return err
			}
			base := os.Getenv("ARENALAB_DRIVER_URL")
			if base == "" {
				base = "http://127.0.0.1:8099"
			}
			r, err := coursegate.New(base, os.Getenv("ARENALAB_TEST_TOKEN"))
			if err != nil {
				return err
			}
			return r.Run(s, os.Stdout)
		}
		tags := "acceptance"
		if referenceMode {
			tags += ",reference"
		}
		cmd := exec.Command("go", "test", "-tags="+tags, "./test/acceptance/rules", "-count=1", "-timeout=30s")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return fmt.Errorf("unknown command")
}

func modelGate() error {
	path := "learning/lab00/model.json"
	if referenceMode {
		path = "reference/model.json"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("fill templates/model.json into learning/lab00/model.json: %w", err)
	}
	var got map[string]any
	if err = json.Unmarshal(b, &got); err != nil {
		return err
	}
	want := map[string]any{
		"committed_ack_requires": "durable_commit",
		"commit_connection_lost": "outcome_unknown",
		"duplicate_command":      "original_result",
		"old_owner":              "reject",
		"realtime_tick_ack":      "volatile_processed",
		"aborted_score":          float64(0),
	}
	if err = coursegate.Match(want, got); err != nil {
		return err
	}
	fmt.Println("PASS: model assertions only; state tables and oral defense still required")
	return nil
}
