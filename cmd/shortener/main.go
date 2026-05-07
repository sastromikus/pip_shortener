package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sastromikus/pip_shortener/internal/audit"
	"github.com/sastromikus/pip_shortener/internal/config"
	"github.com/sastromikus/pip_shortener/internal/handler"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"

	"github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

var buildVersion string
var buildDate string
var buildCommit string

func printBuildInfo() {
	v := buildVersion
	if v == "" {
		v = "N/A"
	}

	d := buildDate
	if d == "" {
		d = "N/A"
	}

	c := buildCommit
	if c == "" {
		c = "N/A"
	}

	fmt.Printf("Build version: %s\n", v)
	fmt.Printf("Build date: %s\n", d)
	fmt.Printf("Build commit: %s\n", c)
}

func migrationPaths() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	candidates := make([]string, 0, 3)

	candidates = append(candidates, filepath.Join(cwd, "migrations"))

	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "migrations"),
			filepath.Clean(filepath.Join(exeDir, "..", "..", "migrations")),
		)
	}

	out := make([]string, 0, len(candidates))
	seen := make(map[string]struct{})
	for _, d := range candidates {
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}

		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			continue
		}
		out = append(out, d)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("migrations directory not found (tried: %v)", candidates)
	}

	return out, nil
}

func runMigrations(db *sql.DB) error {
	dirs, err := migrationPaths()
	if err != nil {
		return fmt.Errorf("migrations: %w", err)
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
			return nil
		}
	}

	if lastErr == nil {
		lastErr = errors.New("no migrations were applied")
	}

	return fmt.Errorf("migrations: %w", lastErr)
}

func main() {
	printBuildInfo()

	var repo repository.URLRepository
	var db *sql.DB
	var observers []audit.Observer

	cfg := config.Parse()
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	if cfg.AuditFile != "" {
		observers = append(observers, audit.NewFileObserver(cfg.AuditFile))
	}
	if cfg.AuditURL != "" {
		observers = append(observers, audit.NewHTTPObserver(cfg.AuditURL))
	}
	auditor := audit.NewNotifier(observers...)

	if cfg.DatabaseDSN != "" {
		d, err := sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("db open: %v", err)
		}
		db = d

		if err := runMigrations(db); err != nil {
			log.Fatal(err)
		}

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
	svc.StartDeleteWorker(128, 500*time.Millisecond)
	router := handler.NewRouter(svc, cfg.BaseURL, logger, db, auditor)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	srvErr := make(chan error, 1)
	go func() {
		var err error
		if cfg.EnableHTTPS {
			const certFile = "server.crt"
			const keyFile = "server.key"
			log.Printf("listening (https) on https://%s\n", cfg.ServerAddr)
			err = srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Printf("listening on http://%s\n", cfg.ServerAddr)
			err = srv.ListenAndServe()
		}
		srvErr <- err
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case sig := <-stop:
		log.Printf("shutdown signal: %v\n", sig)
	case err := <-srvErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)

	err := <-srvErr
	if err != nil && err != http.ErrServerClosed {
		log.Printf("server error after shutdown: %v", err)
	}

	if db != nil {
		_ = db.Close()
	}

	log.Println("shutdown")
}
