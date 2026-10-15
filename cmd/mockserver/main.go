package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

const defaultServiceTime = 20 * time.Millisecond

func workHandler(serviceTime time.Duration, maxInFlight int) http.HandlerFunc {
	var slots chan struct{}
	if maxInFlight > 0 {
		slots = make(chan struct{}, maxInFlight)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		if slots != nil {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			default:
				w.Header().Set("Retry-After", "1")
				http.Error(w, "server overloaded; retry later", http.StatusServiceUnavailable)
				return
			}
		}

		time.Sleep(serviceTime)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "ok")
	}
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address for the mock server to listen on")
	serviceTime := flag.Duration("service-time", defaultServiceTime, "fixed processing time for each request")
	maxInFlight := flag.Int("max-inflight", 0, "maximum concurrent requests; 0 disables overload rejection")
	flag.Parse()

	if *serviceTime < 0 || *maxInFlight < 0 {
		log.Fatal("service-time and max-inflight must not be negative")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("GET /work", workHandler(*serviceTime, *maxInFlight))

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("mock server listening on http://%s/work (service time: %s, max in-flight: %d)", *addr, *serviceTime, *maxInFlight)
	log.Fatal(server.ListenAndServe())
}
