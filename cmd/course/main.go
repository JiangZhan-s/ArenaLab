package main

import (
	"encoding/json"
	"fmt"
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
		return fmt.Errorf("usage: go run ./cmd/course check | gate NN")
	}
	b, err := os.ReadFile("course.json")
	if err != nil {
		return err
	}
	var c manifest
	if err = json.Unmarshal(b, &c); err != nil {
		return err
	}
	if len(c.Labs) != 12 {
		return fmt.Errorf("expected 12 labs")
	}
	if args[0] == "check" {
		seen := map[string]bool{}
		for _, l := range c.Labs {
			if seen[l.ID] {
				return fmt.Errorf("duplicate lab %s", l.ID)
			}
			seen[l.ID] = true
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
	if args[0] == "gate" && len(args) == 2 {
		id := args[1]
		if id == "00" {
			return fmt.Errorf("MANUAL_GATE: submit four state tables, 12 invariants, 10 sequences and oral defense")
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
			return fmt.Errorf("NOT_IMPLEMENTED: Lab %s business runner must be built following its acceptance.json; do not count as passed", id)
		}
		cmd := exec.Command("go", "test", "-tags=acceptance", "./test/acceptance/rules", "-count=1", "-timeout=30s")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return fmt.Errorf("unknown command")
}
