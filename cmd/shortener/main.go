package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"database/sql"
	"path/filepath"

	"github.com/sastromikus/pip_shortener/internal/config"
	"github.com/sastromikus/pip_shortener/internal/handler"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"

    "github.com/sirupsen/logrus"

    _ "github.com/lib/pq"
)

func migrationPaths() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cwdMigrations := filepath.Join(cwd, "migrations")

	exe, err := os.Executable()
	if err != nil {
		return []string{cwdMigrations}, nil
	}
	exeDir := filepath.Dir(exe)

	exeMigrations1 := filepath.Join(exeDir, "migrations")
	exeMigrations2 := filepath.Clean(filepath.Join(exeDir, "..", "..", "migrations"))

	return []string{cwdMigrations, exeMigrations1, exeMigrations2}, nil
}

func runMigrations(db *sql.DB) {
	dirs, err := migrationPaths()
	if err != nil {
		log.Fatalf("migrations: %v", err)
	}

	files := []string{
		"0001_create_urls.sql",
		"0002_unique_original.sql",
		"0003_create_user_urls.sql",
		"0004_add_is_deleted.sql",
	}

	var lastErr error
	for _, dir := range dirs {
		ok := true
		for _, f := range files {
			p := filepath.Join(dir, f)
			if err := repository.RunSQLMigration(db, p); err != nil {
				lastErr = err
				ok = false
				break
			}
		}
		if ok {
			return
		}
	}

	log.Fatalf("migrations: %v", lastErr)
}

func main() {
	var repo repository.URLRepository
	var db *sql.DB

	cfg := config.Parse()
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	if cfg.DatabaseDSN != "" {
		d, err := sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("db open: %v", err)
		}
		db = d

		runMigrations(db)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			log.Printf("db ping failed: %v", err)
		}

		repo = repository.NewPostgresRepository(db)

	} else if cfg.FileStoragePath != "" {
		fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatalf("file repository: %v", err)
		}
		repo = fileRepo
	} else {
		repo = repository.NewMemoryRepository()
	}

	svc := service.NewShortener(repo)
	svc.StartDeleteWorker(128, 500 * time.Millisecond)
	router := handler.NewRouter(svc, cfg.BaseURL, logger, db)

	srv := &http.Server{
	    Addr: cfg.ServerAddr,
	    Handler: router,
	}

	go func() {
		log.Printf("listening on http://%s\n", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if db != nil {
	    _ = db.Close()
	}

	_ = srv.Shutdown(ctx)
	log.Println("shutdown")
}