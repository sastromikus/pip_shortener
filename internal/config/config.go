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

	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
	flag.StringVar(&flagFile, "f", "", "File storage path")
	flag.StringVar(&flagDSN, "d", "", "Database DSN")
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

	if v, ok := os.LookupEnv(envServerAddr); ok && v != "" {
		cfg.ServerAddr = v
	}
	if v, ok := os.LookupEnv(envBaseURL); ok && v != "" {
		cfg.BaseURL = v
	}
	if v, ok := os.LookupEnv(envFileStoragePath); ok && v != "" {
		cfg.FileStoragePath = v
	}
	if v, ok := os.LookupEnv(envDatabaseDSN); ok && v != "" {
		cfg.DatabaseDSN = v
	}

	return cfg
}
