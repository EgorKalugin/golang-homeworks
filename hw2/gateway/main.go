package main

import (
	"context"
	"errors"
	"gateway/internal/api"
	"ledger/app"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc, closeFn, err := app.New(ctx)
	if err != nil {
		log.Fatalf("gateway: cannot start ledger: %v", err)
	}
	defer closeFn()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: api.NewRouter(api.NewHandlers(svc)),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("gateway: shutdown: %v", err)
		}
	}()

	log.Printf("Gateway service started on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("gateway: %v", err)
		return
	}
	log.Println("Gateway service stopped")
}
