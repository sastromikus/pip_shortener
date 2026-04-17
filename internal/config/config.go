package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddr string
	BaseURL    string
}

const (
	defaultServerAddr = "localhost:8080"
	defaultBaseURL    = "http://localhost:8080"

	envServerAddr = "SERVER_ADDRESS"
	envBaseURL    = "BASE_URL"
)

func Parse() Config {
	cfg := Config{
		ServerAddr: defaultServerAddr,
		BaseURL:    defaultBaseURL,
	}

	var flagAddr string
	var flagBase string
	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
	flag.Parse()

	if flagAddr != "" {
		cfg.ServerAddr = flagAddr
	}
	if flagBase != "" {
		cfg.BaseURL = flagBase
	}

	if v := os.Getenv(envServerAddr); v != "" {
		cfg.ServerAddr = v
	}
	if v := os.Getenv(envBaseURL); v != "" {
		cfg.BaseURL = v
	}

	return cfg
}