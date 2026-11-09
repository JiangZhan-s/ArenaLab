//go:build reference

package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/worker"
	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	once := flag.Bool("once", false, "one request")
	match := flag.String("match", "", "match ID")
	winner := flag.String("winner", "", "optional expected winner")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	dsn := os.Getenv("ARENALAB_DSN")
	st, e := store.Open(dsn, "worker")
	if e != nil {
		log.Fatal(e)
	}
	defer st.DB.Close()
	cfg, e := mysql.ParseDSN(dsn)
	if e != nil {
		log.Fatal(e)
	}
	r := redis.NewClient(&redis.Options{Addr: os.Getenv("ARENALAB_REDIS_ADDR")})
	defer r.Close()
	w := worker.Worker{Store: st, Redis: r, Namespace: cfg.DBName}
	if *once {
		var expected *int
		if *winner != "" {
			n, e := strconv.Atoi(*winner)
			if e != nil {
				log.Fatal(e)
			}
			expected = &n
		}
		e = w.Once(ctx, *match, expected)
		out := map[string]any{"error": ""}
		if e != nil {
			out["error"] = e.Error()
		}
		_ = json.NewEncoder(os.Stdout).Encode(out)
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e = w.Pending(ctx); e != nil {
				log.Printf("worker retry: %v", e)
			}
		}
	}
}
