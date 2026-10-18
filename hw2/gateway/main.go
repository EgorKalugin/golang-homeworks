package main

import (
	"gateway/internal/api"
	"log"
	"net/http"
)

func main() {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: api.NewRouter(api.NewHandlers()),
	}

	log.Printf("Gateway service started on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
