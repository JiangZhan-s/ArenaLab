//go:build reference

package main

import (
	"context"
	"flag"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/server"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/store"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("listen", "127.0.0.1:8080", "HTTP address")
	node := flag.String("node", "node1", "unique process owner ID")
	flag.Parse()
	secret := os.Getenv("ARENALAB_DEV_SECRET")
	if len(secret) < 16 {
		log.Fatal("ARENALAB_DEV_SECRET must contain >=16 characters (local development credentials)")
	}
	st, e := store.Open(os.Getenv("ARENALAB_DSN"), *node)
	if e != nil {
		log.Fatal(e)
	}
	defer st.DB.Close()
	if e = st.Migrate(context.Background()); e != nil {
		log.Fatal(e)
	}
	for _, id := range []int{0, 1} {
		if e = st.EnsurePlayer(context.Background(), id); e != nil {
			log.Fatal(e)
		}
	}
	origin := os.Getenv("ARENALAB_ORIGIN")
	if origin == "" {
		origin = "http://localhost:8080"
	}
	s := server.New(st, secret, origin)
	if os.Getenv("ARENALAB_MODE") == "dedicated" {
		s.EnableDedicated()
	}
	if os.Getenv("ARENALAB_MODE") == "realtime" {
		s.EnableRealtime(os.Getenv("ARENALAB_MANUAL_CLOCK") == "1")
	}
	defer s.Close()
	stopHooks, e := installHooks(s)
	if e != nil {
		log.Fatal(e)
	}
	defer stopHooks()
	mux := http.NewServeMux()
	mux.Handle("/", s.Handler())
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	l, e := net.Listen("tcp", *addr)
	if e != nil {
		log.Fatal(e)
	}
	log.Printf("arena node=%s listening=%s", *node, l.Addr())
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-done
		s.Drain.Store(true)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		s.Close()
	}()
	if e = srv.Serve(l); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}
