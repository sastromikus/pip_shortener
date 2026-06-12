package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration derived from defaults, config file, flags and environment variables.
type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	AuditFile       string
	AuditURL        string
	TrustedSubnet   string
	GRPCAddr        string
	EnableHTTPS     bool
}

const (
	defaultServerAddr      = "localhost:8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultFileStoragePath = "storage.json"
	defaultGRPCAddr        = "localhost:3200"

	envServerAddr      = "SERVER_ADDRESS"
	envBaseURL         = "BASE_URL"
	envFileStoragePath = "FILE_STORAGE_PATH"
	envDatabaseDSN     = "DATABASE_DSN"
	envEnableHTTPS     = "ENABLE_HTTPS"
	envAuditFile       = "AUDIT_FILE"
	envAuditURL        = "AUDIT_URL"
	envConfigPath      = "CONFIG"
	envTrustedSubnet   = "TRUSTED_SUBNET"
	envGRPCAddr        = "GRPC_ADDRESS"
)

type fileConfig struct {
	ServerAddr      string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     *bool  `json:"enable_https"`
	AuditFile       string `json:"audit_file"`
	AuditURL        string `json:"audit_url"`
	TrustedSubnet   string `json:"trusted_subnet"`
	GRPCAddr        string `json:"grpc_address"`
}

// Parse reads configuration and applies priority: environment variables > flags > config file > defaults.
func Parse() Config {
	cfg := Config{
		ServerAddr:      defaultServerAddr,
		BaseURL:         defaultBaseURL,
		FileStoragePath: defaultFileStoragePath,
		GRPCAddr:        defaultGRPCAddr,
	}

	var flagAddr string
	var flagBase string
	var flagFile string
	var flagDSN string
	var flagAuditFile string
	var flagAuditURL string
	var flagConfig string
	var flagTrustedSubnet string
	var flagGRPCAddr string
	var flagHTTPS bool

	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
	flag.StringVar(&flagFile, "f", "", "File storage path")
	flag.StringVar(&flagDSN, "d", "", "Database DSN")
	flag.StringVar(&flagDSN, "database-dsn", "", "Database DSN")
	flag.StringVar(&flagDSN, "database_dsn", "", "Database DSN")
	flag.StringVar(&flagAuditFile, "audit-file", "", "Audit log file path")
	flag.StringVar(&flagAuditURL, "audit-url", "", "Audit receiver URL")
	flag.BoolVar(&flagHTTPS, "s", false, "Enable HTTPS")
	flag.StringVar(&flagTrustedSubnet, "t", "", "Trusted subnet CIDR")
	flag.StringVar(&flagGRPCAddr, "g", "", "gRPC server address")
	flag.StringVar(&flagGRPCAddr, "grpc-address", "", "gRPC server address")
	flag.StringVar(&flagConfig, "c", "", "Config file path")
	flag.StringVar(&flagConfig, "config", "", "Config file path")
	flag.Parse()

	flagsSet := visitedFlags()

	configPath := flagConfig
	if v, ok := os.LookupEnv(envConfigPath); ok {
		configPath = v
	}

	if fc, err := readConfigFile(configPath); err == nil {
		applyFileConfig(&cfg, fc)
	}

	if flagsSet["a"] {
		cfg.ServerAddr = flagAddr
	}
	if flagsSet["b"] {
		cfg.BaseURL = flagBase
	}
	if flagsSet["f"] {
		cfg.FileStoragePath = flagFile
	}
	if flagsSet["d"] || flagsSet["database-dsn"] || flagsSet["database_dsn"] {
		cfg.DatabaseDSN = flagDSN
	}
	if flagsSet["audit-file"] {
		cfg.AuditFile = flagAuditFile
	}
	if flagsSet["audit-url"] {
		cfg.AuditURL = flagAuditURL
	}
	if flagsSet["s"] {
		cfg.EnableHTTPS = flagHTTPS
	}
	if flagsSet["t"] {
		cfg.TrustedSubnet = flagTrustedSubnet
	}
	if flagsSet["g"] || flagsSet["grpc-address"] {
		cfg.GRPCAddr = flagGRPCAddr
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
	if v, ok := os.LookupEnv(envEnableHTTPS); ok {
		if enableHTTPS, err := strconv.ParseBool(v); err == nil {
			cfg.EnableHTTPS = enableHTTPS
		}
	}
	if v, ok := os.LookupEnv(envTrustedSubnet); ok {
		cfg.TrustedSubnet = v
	}
	if v, ok := os.LookupEnv(envGRPCAddr); ok {
		cfg.GRPCAddr = v
	}

	return cfg
}

func visitedFlags() map[string]bool {
	out := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		out[f.Name] = true
	})
	return out
}

func readConfigFile(path string) (fileConfig, error) {
	if strings.TrimSpace(path) == "" {
		return fileConfig{}, os.ErrNotExist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, err
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, err
	}

	return cfg, nil
}

func applyFileConfig(cfg *Config, fc fileConfig) {
	if fc.ServerAddr != "" {
		cfg.ServerAddr = fc.ServerAddr
	}
	if fc.BaseURL != "" {
		cfg.BaseURL = fc.BaseURL
	}
	if fc.FileStoragePath != "" {
		cfg.FileStoragePath = fc.FileStoragePath
	}
	if fc.DatabaseDSN != "" {
		cfg.DatabaseDSN = fc.DatabaseDSN
	}
	if fc.AuditFile != "" {
		cfg.AuditFile = fc.AuditFile
	}
	if fc.AuditURL != "" {
		cfg.AuditURL = fc.AuditURL
	}
	if fc.EnableHTTPS != nil {
		cfg.EnableHTTPS = *fc.EnableHTTPS
	}
	if fc.TrustedSubnet != "" {
		cfg.TrustedSubnet = fc.TrustedSubnet
	}
	if fc.GRPCAddr != "" {
		cfg.GRPCAddr = fc.GRPCAddr
	}
}
