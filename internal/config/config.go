package config

import "flag"

type Config struct {
	ServerAddr string
	BaseURL    string
}

func Parse() Config {
	cfg := Config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "Base URL for short links")

	flag.Parse()
	
	return cfg
}