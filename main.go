package main

import (
	"log"
	"net/http"
	"time"

	"github.com/igorsilva-dev/crashdummy/app/handlers"
	"github.com/igorsilva-dev/crashdummy/app/metrics"
)

func main() {
	mux := http.NewServeMux()
	if err := handlers.Register(mux); err != nil {
		log.Fatalf("loading configuration: %v", err)
	}
	mux.Handle("GET /metrics", metrics.Handler())

	server := &http.Server{
		Addr:              ":10000",
		Handler:           metrics.Instrument(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Println("crashdummy listening at :10000")
	log.Fatal(server.ListenAndServe())
}
