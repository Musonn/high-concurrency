package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

const defaultServiceTime = 20 * time.Millisecond

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address for the mock server to listen on")
	serviceTime := flag.Duration("service-time", defaultServiceTime, "fixed processing time for each request")
	flag.Parse()
	if *serviceTime < 0 {
		log.Fatal("service-time must not be negative")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /work", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(*serviceTime)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("mock server listening on http://%s/work (fixed service time: %s)", *addr, *serviceTime)
	log.Fatal(server.ListenAndServe())
}
