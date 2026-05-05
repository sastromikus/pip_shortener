package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	AuditFile       string
	AuditURL        string
}

const (
	defaultServerAddr = "localhost:8080"
	defaultBaseURL    = "http://localhost:8080"

	envServerAddr = "SERVER_ADDRESS"
	envBaseURL    = "BASE_URL"

	defaultFileStoragePath = "storage.json"
	envFileStoragePath     = "FILE_STORAGE_PATH"

	envDatabaseDSN = "DATABASE_DSN"
)

func Parse() Config {
	cfg := Config{
		ServerAddr:      defaultServerAddr,
		BaseURL:         defaultBaseURL,
		FileStoragePath: defaultFileStoragePath,
	}

	var flagAddr string
	var flagBase string
	var flagFile string
	var flagDSN string
	var flagAuditFile string
	var flagAuditURL string

	flag.StringVar(&flagFile, "f", "", "File storage path")
	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
	flag.StringVar(&flagDSN, "d", "", "Database DSN")
	flag.StringVar(&flagDSN, "database-dsn", "", "Database DSN")
	flag.StringVar(&flagDSN, "database_dsn", "", "Database DSN")
	flag.StringVar(&flagAuditFile, "audit-file", "", "Audit log file path")
	flag.StringVar(&flagAuditURL, "audit-url", "", "Audit receiver URL")
	flag.Parse()

	if flagAddr != "" {
		cfg.ServerAddr = flagAddr
	}
	if flagBase != "" {
		cfg.BaseURL = flagBase
	}
	if flagFile != "" {
		cfg.FileStoragePath = flagFile
	}
	if flagDSN != "" {
		cfg.DatabaseDSN = flagDSN
	}
	if flagAuditFile != "" {
		cfg.AuditFile = flagAuditFile
	}
	if flagAuditURL != "" {
		cfg.AuditURL = flagAuditURL
	}

	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}
	if v := os.Getenv(envServerAddr); v != "" {
		cfg.ServerAddr = v
	}
	if v := os.Getenv(envBaseURL); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv(envFileStoragePath); v != "" {
		cfg.FileStoragePath = v
	}
	if v := os.Getenv(envDatabaseDSN); v != "" {
		cfg.DatabaseDSN = v
	}
	if v := os.Getenv("AUDIT_FILE"); v != "" {
		cfg.AuditFile = v
	}
	if v := os.Getenv("AUDIT_URL"); v != "" {
		cfg.AuditURL = v
	}

	return cfg
}
