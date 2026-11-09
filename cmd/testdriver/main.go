//go:build reference

package main

import (
	"flag"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/testdriver"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("listen", "127.0.0.1:8099", "loopback only")
	binary := flag.String("arena", "bin/arena-server", "reference/testhooks binary")
	flag.Parse()
	host, _, e := net.SplitHostPort(*addr)
	if e != nil || host != "127.0.0.1" {
		log.Fatal("driver requires literal loopback")
	}
	token := os.Getenv("ARENALAB_TEST_TOKEN")
	secret := os.Getenv("ARENALAB_DEV_SECRET")
	if len(token) < 16 || len(secret) < 16 {
		log.Fatal("test token and dev secret must contain >=16 characters")
	}
	path, e := testdriver.ResolveBinary(*binary)
	if e != nil {
		log.Fatal(e)
	}
	d := testdriver.New(os.Getenv("ARENALAB_ADMIN_DSN"), path, secret, token, "artifacts/raw/reference")
	defer d.Close()
	srv := &http.Server{Addr: *addr, Handler: d.Handler(), ReadHeaderTimeout: 3 * time.Second}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() { <-done; _ = srv.Close() }()
	log.Printf("driver listening=%s", *addr)
	if e = srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}
