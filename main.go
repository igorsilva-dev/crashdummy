package main

import (
	"log"
	"net/http"
	"time"

	"github.com/igorsilva-dev/crashdummy/app/handlers"
)

func main() {
	mux := http.NewServeMux()
	if err := handlers.Register(mux); err != nil {
		log.Fatalf("loading configuration: %v", err)
	}

	server := &http.Server{
		Addr:              ":10000",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Println("crashdummy listening at :10000")
	log.Fatal(server.ListenAndServe())
}
