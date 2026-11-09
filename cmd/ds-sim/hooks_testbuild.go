//go:build reference && testhooks

package main

import (
	"encoding/json"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"net/http"
	"os"
)

func installTestRoutes(mux *http.ServeMux, r *dedicated.Runtime) {
	mux.HandleFunc("POST /test/result", func(w http.ResponseWriter, q *http.Request) {
		if q.Header.Get("Authorization") != "Bearer "+os.Getenv("ARENALAB_TEST_TOKEN") {
			http.Error(w, "forbidden", 403)
			return
		}
		var v struct {
			Allocation string `json:"allocation_id"`
			Generation int    `json:"generation"`
			Winner     int    `json:"winner"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, q.Body, 8192)).Decode(&v) != nil {
			http.Error(w, "invalid", 400)
			return
		}
		out, e := r.Submit(q.Context(), v.Allocation, v.Generation, v.Winner)
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(out)
	})
}
