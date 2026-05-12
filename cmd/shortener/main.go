package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sastromikus/pip_shortener/internal/config"
	"github.com/sastromikus/pip_shortener/internal/handler"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Parse()

	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)
	router := handler.NewRouter(svc, cfg.BaseURL)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("listening on http://%s\n", cfg.ServerAddr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
		return err
	}

	log.Println("shutdown")
	return nil
}
