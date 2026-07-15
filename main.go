package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/igorsilva-dev/crashdummy/app/handlers"
	"github.com/igorsilva-dev/crashdummy/app/metrics"
)

const defaultPort = "10000"

func main() {
	if dir := os.Getenv("CRASHDUMMY_CONFIG_DIR"); dir != "" {
		handlers.SetConfigDir(dir)
	}

	mux := http.NewServeMux()
	if err := handlers.Register(mux); err != nil {
		log.Fatalf("loading configuration: %v", err)
	}
	mux.Handle("GET /metrics", metrics.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	addr := ":" + port

	server := &http.Server{
		Addr:              addr,
		Handler:           metrics.Instrument(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// #nosec G706 -- addr is built from the operator-set PORT env var, not
	// request input.
	log.Printf("crashdummy listening at %s", addr)
	log.Fatal(server.ListenAndServe())
}
