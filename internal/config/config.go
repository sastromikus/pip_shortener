package config

import (
	"flag"
	"os"
)

// Config holds application configuration derived from flags and environment variables.
type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	AuditFile       string
	AuditURL        string
}

const (
	defaultServerAddr      = "localhost:8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultFileStoragePath = "storage.json"

	envServerAddr      = "SERVER_ADDRESS"
	envBaseURL         = "BASE_URL"
	envFileStoragePath = "FILE_STORAGE_PATH"
	envDatabaseDSN     = "DATABASE_DSN"
	envAuditFile       = "AUDIT_FILE"
	envAuditURL        = "AUDIT_URL"
)

// Parse reads flags and environment variables and returns the resulting configuration.
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

	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
	flag.StringVar(&flagFile, "f", "", "File storage path")
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

	if v, ok := os.LookupEnv(envServerAddr); ok {
		cfg.ServerAddr = v
	}
	if v, ok := os.LookupEnv(envBaseURL); ok {
		cfg.BaseURL = v
	}
	if v, ok := os.LookupEnv(envFileStoragePath); ok {
		cfg.FileStoragePath = v
	}
	if v, ok := os.LookupEnv(envDatabaseDSN); ok {
		cfg.DatabaseDSN = v
	}
	if v, ok := os.LookupEnv(envAuditFile); ok {
		cfg.AuditFile = v
	}
	if v, ok := os.LookupEnv(envAuditURL); ok {
		cfg.AuditURL = v
	}

	return cfg
}
