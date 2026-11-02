package main

import (
	"context"
	"gateway/internal/api"
	"ledger/transactions"
	"log"
	"net/http"
)

func main() {
	if err := transactions.Init(context.Background()); err != nil {
		log.Fatalf("gateway: cannot start ledger storage: %v", err)
	}
	defer transactions.Close()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: api.NewRouter(api.NewHandlers()),
	}

	log.Printf("Gateway service started on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
