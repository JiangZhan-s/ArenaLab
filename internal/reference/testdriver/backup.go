//go:build reference

package testdriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type tableImage struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

var backupTables = []string{"players", "matches", "commands", "events", "results", "ledger", "ratings", "tickets", "occupancy"}

func (d *Driver) backup(ctx context.Context, a map[string]any) (map[string]any, error) {
	switch str(a, "action", "") {
	case "snapshot":
		tx, e := d.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		images := []tableImage{}
		for _, name := range backupTables {
			rows, e := tx.QueryContext(ctx, "SELECT * FROM "+name)
			if e != nil {
				return nil, e
			}
			cols, e := rows.Columns()
			if e != nil {
				rows.Close()
				return nil, e
			}
			im := tableImage{Name: name, Columns: cols}
			for rows.Next() {
				values := make([]any, len(cols))
				ptr := make([]any, len(cols))
				for i := range ptr {
					ptr[i] = &values[i]
				}
				if e = rows.Scan(ptr...); e != nil {
					rows.Close()
					return nil, e
				}
				for i, v := range values {
					switch x := v.(type) {
					case []byte:
						values[i] = string(x)
					case time.Time:
						values[i] = x.UTC().Format("2006-01-02 15:04:05.999999")
					}
				}
				im.Rows = append(im.Rows, values)
			}
			e = rows.Err()
			rows.Close()
			if e != nil {
				return nil, e
			}
			images = append(images, im)
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
		f, e := os.CreateTemp(d.Artifacts, "consistent-backup-*.json")
		if e != nil {
			return nil, e
		}
		e = json.NewEncoder(f).Encode(images)
		closeErr := f.Close()
		if e != nil {
			return nil, e
		}
		if closeErr != nil {
			return nil, closeErr
		}
		d.backupFile = f.Name()
		return map[string]any{"saved": true}, nil
	case "restore_to_new_database":
		if d.backupFile == "" {
			return nil, fmt.Errorf("no snapshot")
		}
		b, e := os.ReadFile(d.backupFile)
		if e != nil {
			return nil, e
		}
		var images []tableImage
		if e = json.Unmarshal(b, &images); e != nil {
			return nil, e
		}
		target := New(d.AdminDSN, d.Binary, d.Secret, d.Token, d.Artifacts)
		if _, e = target.resetState(ctx, map[string]any{"profile": "turn", "seed": "empty"}); e != nil {
			target.Close()
			return nil, e
		}
		target.stop(target.processes["node1"])
		tx, e := target.DB.BeginTx(ctx, nil)
		if e != nil {
			target.Close()
			return nil, e
		}
		defer tx.Rollback()
		if _, e = tx.ExecContext(ctx, "DELETE FROM players"); e != nil {
			target.Close()
			return nil, e
		}
		for _, im := range images {
			allowed := false
			for _, n := range backupTables {
				if n == im.Name {
					allowed = true
				}
			}
			if !allowed {
				target.Close()
				return nil, fmt.Errorf("unknown backup table")
			}
			for _, c := range im.Columns {
				if strings.ContainsAny(c, "` ;\n") {
					target.Close()
					return nil, fmt.Errorf("invalid column")
				}
			}
			marks := strings.TrimSuffix(strings.Repeat("?,", len(im.Columns)), ",")
			q := "INSERT INTO " + im.Name + " (`" + strings.Join(im.Columns, "`,`") + "`) VALUES(" + marks + ")"
			for _, row := range im.Rows {
				if _, e = tx.ExecContext(ctx, q, row...); e != nil {
					target.Close()
					return nil, e
				}
			}
		}
		if e = tx.Commit(); e != nil {
			target.Close()
			return nil, e
		}
		if _, e = target.start("node1"); e != nil {
			target.Close()
			return nil, e
		}
		target.match = d.match
		d.restored = target
		return map[string]any{"restored": true}, nil
	default:
		return nil, fmt.Errorf("unknown backup operation")
	}
}
