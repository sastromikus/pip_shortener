package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sastromikus/pip_shortener/internal/config"
	"github.com/sastromikus/pip_shortener/internal/handler"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Parse()

	logger := slog.Default()

	var (
		repo service.URLRepository
		db   *sql.DB
	)

	if cfg.DatabaseDSN != "" {
		dbCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		postgresDB, postgresRepo, err := repository.NewPostgresStorage(dbCtx, cfg.DatabaseDSN)
		if err != nil {
			return err
		}

		db = postgresDB
		defer db.Close()

		repo = postgresRepo
	} else if cfg.FileStoragePath != "" {
		fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
		if err != nil {
			return fmt.Errorf("file repository: %w", err)
		}
		repo = fileRepo
	} else {
		repo = repository.NewMemoryRepository()
	}

	svc := service.NewShortener(repo)
	svc.StartDeleteWorker(128, 500*time.Millisecond)

	router := handler.NewRouter(svc, cfg.BaseURL, logger, db)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	serverErr := make(chan error, 1)

	go func() {
		logger.Info("listening on", "addr", cfg.ServerAddr)

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
		logger.Error("server shutdown error", "error", err)
		return err
	}

	logger.Info("shutdown")
	return nil
}
