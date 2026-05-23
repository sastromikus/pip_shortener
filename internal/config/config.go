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
}

const (
	defaultServerAddr      = "localhost:8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultFileStoragePath = "storage.json"

	envServerAddr      = "SERVER_ADDRESS"
	envBaseURL         = "BASE_URL"
	envFileStoragePath = "FILE_STORAGE_PATH"
	envDatabaseDSN     = "DATABASE_DSN"
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

	flag.StringVar(&flagFile, "f", "", "File storage path")
	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
	flag.StringVar(&flagDSN, "d", "", "Database DSN")
	flag.StringVar(&flagDSN, "database-dsn", "", "Database DSN")
	flag.StringVar(&flagDSN, "database_dsn", "", "Database DSN")
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

	return cfg
}
