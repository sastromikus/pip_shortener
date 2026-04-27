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

	"github.com/sastromikus/pip_shortener/internal/config"
	"github.com/sastromikus/pip_shortener/internal/handler"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"

    "github.com/sirupsen/logrus"

    _ "github.com/lib/pq"
)

func main() {
	var repo repository.URLRepository
	var db *sql.DB

	cfg := config.Parse()
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
	if err != nil {
	    log.Fatalf("file repository: %v", err)
	}
	repo = fileRepo

	if cfg.DatabaseDSN != "" {
	    d, err := sql.Open("postgres", cfg.DatabaseDSN)
	    if err != nil { log.Fatalf("db open: %v", err) }
	    db = d

	    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	    defer cancel()
	    if err := db.PingContext(ctx); err != nil {
	        log.Printf("db ping failed: %v", err)
	    }
	}

	svc := service.NewShortener(repo)
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

	_ = srv.Shutdown(ctx)
	log.Println("shutdown")
}