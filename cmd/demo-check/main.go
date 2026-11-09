//go:build reference

// Independently audit the browser journey against MySQL, not UI counters.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var report struct {
		Passed   bool `json:"passed"`
		Journeys []struct {
			Mode      string `json:"mode"`
			Snapshots []struct {
				Match string `json:"match"`
			} `json:"snapshots"`
		} `json:"journeys"`
	}
	b, e := os.ReadFile("artifacts/raw/browser/report.json")
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, &report); e != nil {
		return e
	}
	if !report.Passed || len(report.Journeys) != 2 {
		return fmt.Errorf("two passed browser journeys required")
	}
	db, e := sql.Open("mysql", os.Getenv("ARENALAB_DSN"))
	if e != nil {
		return e
	}
	defer db.Close()
	evidence := []map[string]any{}
	for _, j := range report.Journeys {
		if len(j.Snapshots) != 2 || j.Snapshots[0].Match != j.Snapshots[1].Match {
			return fmt.Errorf("client match disagreement")
		}
		id := j.Snapshots[0].Match
		var state string
		var ledger, results int
		deadline := time.Now().Add(5 * time.Second)
		for {
			e = db.QueryRow("SELECT state FROM results WHERE match_id=?", id).Scan(&state)
			if e == nil && state == "applied" {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("%s not settled: %s %v", id, state, e)
			}
			time.Sleep(50 * time.Millisecond)
		}
		if e = db.QueryRow("SELECT COUNT(*) FROM results WHERE match_id=?", id).Scan(&results); e != nil {
			return e
		}
		if e = db.QueryRow("SELECT COUNT(*) FROM ledger WHERE match_id=?", id).Scan(&ledger); e != nil {
			return e
		}
		if results != 1 || ledger != 2 {
			return fmt.Errorf("%s result=%d ledger=%d", id, results, ledger)
		}
		var bad int
		if e = db.QueryRow("SELECT COUNT(*) FROM ledger l JOIN results r ON r.match_id=l.match_id JOIN matches m ON m.id=l.match_id WHERE l.match_id=? AND l.delta<>CASE WHEN r.winner=-1 THEN 0 WHEN (r.winner=0 AND l.player_id=m.p0) OR (r.winner=1 AND l.player_id=m.p1) THEN 10 ELSE -10 END", id).Scan(&bad); e != nil {
			return e
		}
		if bad != 0 {
			return fmt.Errorf("wrong score for %s", id)
		}
		evidence = append(evidence, map[string]any{"mode": j.Mode, "match": id, "results": results, "ledger_rows": ledger, "state": state, "score_matches_winner": true})
	}
	b, e = json.MarshalIndent(evidence, "", "  ")
	if e != nil {
		return e
	}
	if e = os.WriteFile("artifacts/raw/browser/database-check.json", b, 0600); e != nil {
		return e
	}
	fmt.Println("PASS browser SQL audit: unique terminal results, two ledger rows per match, winner-to-score mapping")
	return nil
}
