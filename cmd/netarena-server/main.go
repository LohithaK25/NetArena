package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"example.com/netarena/internal/api"
)

func main() {
	addr := os.Getenv("NETARENA_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("NetArena API listening on http://%s", addr)
	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("NetArena API server failed: %v", err)
	}
}
