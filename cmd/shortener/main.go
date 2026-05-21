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

	_ "github.com/lib/pq"

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
		d, err := sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("db open: %w", err)
		}
		db = d
		defer db.Close()

		if err := repository.RunSQLMigration(db, "migrations/0001_create_urls.sql"); err != nil {
			return fmt.Errorf("migrations 0001_create_urls: %w", err)
		}
		if err := repository.RunSQLMigration(db, "migrations/0002_unique_original.sql"); err != nil {
			return fmt.Errorf("migrations 0002_unique_original: %w", err)
		}

		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := db.PingContext(pingCtx); err != nil {
			cancel()
			return fmt.Errorf("db ping: %w", err)
		}
		cancel()

		repo = repository.NewPostgresRepository(db)
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
