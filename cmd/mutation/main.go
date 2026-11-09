// Command mutation runs deliberate source mutations in disposable copies.
// It never edits the learner workspace or the fixed oracle.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type mutation struct{ name, file, before, after, gate, expected string }

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if os.Getenv("ARENALAB_ADMIN_DSN") == "" {
		return fmt.Errorf("isolated verification MySQL required")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	cases := []mutation{
		{"ack-without-commit", "internal/reference/store/store.go", "if e = tx.Commit(); e != nil {\n\t\treturn Reply{Error: \"outcome_unknown\"}, e\n\t}", "// MUTANT: return durable success without a commit.", "04", "FAIL 04/dedup-before-version"},
		{"version-before-dedup", "internal/reference/store/store.go", "payload, _ := json.Marshal(c)", "if c.Version != m.State.Version { return fail(\"stale_version\") }; payload, _ := json.Marshal(c)", "04", "FAIL 04/dedup-before-version"},
		{"missing-fence", "internal/reference/store/store.go", "if m.Owner != s.Node || m.Epoch != ownerEpoch || !now.Before(m.Lease) {", "if false && (m.Owner != s.Node || m.Epoch != ownerEpoch || !now.Before(m.Lease)) {", "07", "FAIL 07/fenced-zombie-owner"},
		{"settlement-not-idempotent", "internal/reference/store/store.go", "if state == \"applied\" {", "if false && state == \"applied\" {", "08", "FAIL 08/settlement-replay-and-conflict"},
		{"expired-ticket-accepted", "internal/reference/dedicated/control.go", "if t.Expires <= time.Now().Unix() {", "if false && t.Expires <= time.Now().Unix() {", "13", "FAIL 13/expired-ticket-wrong-build-and-player-forgery"},
	}
	destination := filepath.Join(root, "artifacts/raw/mutations")
	if err = os.MkdirAll(destination, 0700); err != nil {
		return err
	}
	for _, m := range cases {
		dir, e := os.MkdirTemp("", "arenalab-mutation-")
		if e != nil {
			return e
		}
		fmt.Printf("mutation %s isolated copy %s\n", m.name, dir)
		e = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, _ := filepath.Rel(root, path)
			if d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", "artifacts", "bin":
					return filepath.SkipDir
				}
				return os.MkdirAll(filepath.Join(dir, rel), 0755)
			}
			if !d.Type().IsRegular() {
				return nil
			}
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			return os.WriteFile(filepath.Join(dir, rel), b, 0644)
		})
		if e != nil {
			return e
		}
		path := filepath.Join(dir, m.file)
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		src := strings.ReplaceAll(string(b), "\r\n", "\n")
		if strings.Count(src, m.before) != 1 {
			return fmt.Errorf("%s source anchor changed", m.name)
		}
		if e = os.WriteFile(path, []byte(strings.Replace(src, m.before, m.after, 1)), 0644); e != nil {
			return e
		}
		for _, target := range []string{"arena-server", "settlement-worker", "ds-sim"} {
			cmd := exec.Command("go", "build", "-race", "-tags", "reference,testhooks", "-o", filepath.Join(dir, "bin", target), "./cmd/"+target)
			cmd.Dir = dir
			out, e := cmd.CombinedOutput()
			if e != nil {
				return fmt.Errorf("mutation compile is NOT a kill: %s: %w\n%s", m.name, e, out)
			}
		}
		cmd := exec.Command("go", "test", "-race", "-tags", "reference,e2e", "./internal/reference/testdriver", "-run", "^TestRealProcessGates$", "-count=1", "-v", "-timeout=3m")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "ARENALAB_GATES="+m.gate)
		out, e := cmd.CombinedOutput()
		if err = os.WriteFile(filepath.Join(destination, m.name+".log"), out, 0600); err != nil {
			return err
		}
		if e == nil || !strings.Contains(string(out), m.expected) {
			return fmt.Errorf("mutant %s NOT killed by intended assertion; inspect log (error=%v)", m.name, e)
		}
		fmt.Printf("KILLED %s: %s\n", m.name, m.expected)
		// Retain the exact disposable source and process artifacts for inspection.
	}
	return nil
}
