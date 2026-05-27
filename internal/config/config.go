package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
}

const (
	defaultServerAddr = "localhost:8080"
	defaultBaseURL    = "http://localhost:8080"

	envServerAddr = "SERVER_ADDRESS"
	envBaseURL    = "BASE_URL"

	defaultFileStoragePath = "storage.json"
	envFileStoragePath     = "FILE_STORAGE_PATH"
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

	flag.StringVar(&flagFile, "f", "", "File storage path")
	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
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

	if v := os.Getenv(envServerAddr); v != "" {
		cfg.ServerAddr = v
	}
	if v := os.Getenv(envBaseURL); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv(envFileStoragePath); v != "" {
		cfg.FileStoragePath = v
	}

	return cfg
}
