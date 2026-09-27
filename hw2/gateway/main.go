package main

import (
	"fmt"
	"log"
	"net/http"
)

func ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("pong"))
}

func main() {
	fmt.Println("Gateway service started")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", ping)


	log.Fatal(http.ListenAndServe(":8080", mux))
}
