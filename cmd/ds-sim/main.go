//go:build reference

package main

import (
	"flag"
	"github.com/JiangZhan-s/ArenaLab/internal/reference/dedicated"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("listen", "127.0.0.1:8101", "listen")
	id := flag.String("id", "ds1", "instance identity")
	base := flag.String("backend", "http://127.0.0.1:8080", "Go backend")
	flag.Parse()
	auth, ticket := os.Getenv("ARENALAB_DS_AUTH"), os.Getenv("ARENALAB_DS_TICKET_KEY")
	if len(auth) < 32 || len(ticket) < 32 {
		log.Fatal("DS keys required")
	}
	l, e := net.Listen("tcp", *addr)
	if e != nil {
		log.Fatal(e)
	}
	r := dedicated.NewRuntime(*id, *base, "http://"+l.Addr().String(), auth, ticket)
	if origin := os.Getenv("ARENALAB_ORIGIN"); origin != "" {
		r.Origin = origin
	}
	defer r.Close()
	mux := http.NewServeMux()
	mux.Handle("/", r.Handler())
	installTestRoutes(mux, r)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go srv.Serve(l)
	if e = r.Start(); e != nil {
		log.Fatal(e)
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	<-done
	_ = srv.Close()
}
