package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sastromikus/pip_shortener/internal/audit"
	"github.com/sastromikus/pip_shortener/internal/config"
	"github.com/sastromikus/pip_shortener/internal/handler"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
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

func main() {
	printBuildInfo()

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

	observers := make([]audit.Observer, 0, 2)
	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return fmt.Errorf("audit file observer: %w", err)
		}
		observers = append(observers, fileObserver)
	}
	if cfg.AuditURL != "" {
		observers = append(observers, audit.NewHTTPObserver(cfg.AuditURL))
	}
	auditor := audit.NewNotifier(observers...)
	defer func() {
		if err := auditor.Close(); err != nil {
			logger.Error("audit shutdown failed", "error", err)
		}
	}()

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

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	waitDeleteWorker := svc.StartDeleteWorker(workerCtx, 128, 500*time.Millisecond, func(err error) {
		logger.Error("delete worker failed", "error", err)
	})

	router := handler.NewRouter(svc, cfg.BaseURL, logger, db, auditor)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.ServerAddr, "https", cfg.EnableHTTPS)
		serverErr <- serveHTTP(srv, cfg)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	var runErr error
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		runErr = srv.Shutdown(shutdownCtx)
		cancel()
		if runErr != nil {
			logger.Error("server shutdown error", "error", runErr)
		}
	case err := <-serverErr:
		if err != nil {
			runErr = fmt.Errorf("serve HTTP: %w", err)
		}
	}

	workerCancel()
	waitDeleteWorker()

	if runErr != nil {
		return runErr
	}

	logger.Info("shutdown")
	return nil
}

func serveHTTP(srv *http.Server, cfg config.Config) error {
	if !cfg.EnableHTTPS {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}

	tlsConfig, err := selfSignedTLSConfig(cfg.ServerAddr)
	if err != nil {
		return err
	}

	ln, err := tls.Listen("tcp", cfg.ServerAddr, tlsConfig)
	if err != nil {
		return err
	}

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func selfSignedTLSConfig(hostport string) (*tls.Config, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate tls key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, fmt.Errorf("generate tls serial: %w", err)
	}

	notBefore := time.Now().Add(-time.Hour)
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"pip_shortener"},
		},
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	if host == "" || host == "localhost" {
		tmpl.DNSNames = append(tmpl.DNSNames, "localhost")
	}
	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
	} else if host != "" {
		tmpl.DNSNames = append(tmpl.DNSNames, host)
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("create tls certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load tls key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
