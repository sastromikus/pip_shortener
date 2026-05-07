package config

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
)

// Config holds application configuration derived from flags, config file and environment variables.
type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	AuditFile       string
	AuditURL        string
	EnableHTTPS     bool
}

const (
	defaultServerAddr = "localhost:8080"
	defaultBaseURL    = "http://localhost:8080"

	envServerAddr = "SERVER_ADDRESS"
	envBaseURL    = "BASE_URL"

	defaultFileStoragePath = "storage.json"
	envFileStoragePath     = "FILE_STORAGE_PATH"

	envDatabaseDSN = "DATABASE_DSN"
	envEnableHTTPS = "ENABLE_HTTPS"
	envAuditFile   = "AUDIT_FILE"
	envAuditURL    = "AUDIT_URL"

	envConfigPath = "CONFIG"
)

type fileConfig struct {
	ServerAddr      string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     *bool  `json:"enable_https"`
	AuditFile       string `json:"audit_file"`
	AuditURL        string `json:"audit_url"`
}

// Parse reads flags, config file and environment variables and returns the resulting configuration.
// Priority: config file (low) < flags < environment variables (high).
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
	var flagHTTPS bool
	var flagConfig string

	flag.StringVar(&flagFile, "f", "", "File storage path")
	flag.StringVar(&flagAddr, "a", "", "HTTP server address")
	flag.StringVar(&flagBase, "b", "", "Base URL for short links")
	flag.StringVar(&flagDSN, "d", "", "Database DSN")
	flag.StringVar(&flagDSN, "database-dsn", "", "Database DSN")
	flag.StringVar(&flagDSN, "database_dsn", "", "Database DSN")
	flag.StringVar(&flagAuditFile, "audit-file", "", "Audit log file path")
	flag.StringVar(&flagAuditURL, "audit-url", "", "Audit receiver URL")
	flag.BoolVar(&flagHTTPS, "s", false, "Enable HTTPS")

	flag.StringVar(&flagConfig, "c", "", "Config file path (JSON)")
	flag.StringVar(&flagConfig, "config", "", "Config file path (JSON)")

	flag.Parse()

	flagsSet := struct {
		addrSet   bool
		baseSet   bool
		fileSet   bool
		dsnSet    bool
		httpsSet  bool
		auditFSet bool
		auditUSet bool
	}{
		addrSet:   flagAddr != "",
		baseSet:   flagBase != "",
		fileSet:   flagFile != "",
		dsnSet:    flagDSN != "",
		httpsSet:  flagHTTPS,
		auditFSet: flagAuditFile != "",
		auditUSet: flagAuditURL != "",
	}

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
	if flagHTTPS {
		cfg.EnableHTTPS = true
	}

	configPath := flagConfig
	if v := os.Getenv(envConfigPath); v != "" {
		configPath = v
	}

	if fc, err := readConfigFile(configPath); err == nil {
		if !flagsSet.addrSet && cfg.ServerAddr == defaultServerAddr && fc.ServerAddr != "" {
			cfg.ServerAddr = fc.ServerAddr
		}
		if !flagsSet.baseSet && cfg.BaseURL == defaultBaseURL && fc.BaseURL != "" {
			cfg.BaseURL = fc.BaseURL
		}
		if !flagsSet.fileSet && cfg.FileStoragePath == defaultFileStoragePath && fc.FileStoragePath != "" {
			cfg.FileStoragePath = fc.FileStoragePath
		}
		if !flagsSet.dsnSet && cfg.DatabaseDSN == "" && fc.DatabaseDSN != "" {
			cfg.DatabaseDSN = fc.DatabaseDSN
		}
		if !flagsSet.httpsSet && !cfg.EnableHTTPS && fc.EnableHTTPS != nil {
			cfg.EnableHTTPS = *fc.EnableHTTPS
		}
		if !flagsSet.auditFSet && cfg.AuditFile == "" && fc.AuditFile != "" {
			cfg.AuditFile = fc.AuditFile
		}
		if !flagsSet.auditUSet && cfg.AuditURL == "" && fc.AuditURL != "" {
			cfg.AuditURL = fc.AuditURL
		}
	}

	if v := os.Getenv(envDatabaseDSN); v != "" {
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
	if v := os.Getenv(envAuditFile); v != "" {
		cfg.AuditFile = v
	}
	if v := os.Getenv(envAuditURL); v != "" {
		cfg.AuditURL = v
	}
	if v := os.Getenv(envEnableHTTPS); v != "" {
		cfg.EnableHTTPS = parseEnvBool(v)
	}

	return cfg
}

func readConfigFile(path string) (fileConfig, error) {
	var fc fileConfig
	if strings.TrimSpace(path) == "" {
		return fc, os.ErrNotExist
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fc, err
	}
	if err := json.Unmarshal(b, &fc); err != nil {
		return fc, err
	}
	return fc, nil
}

func parseEnvBool(v string) bool {
	s := strings.TrimSpace(strings.ToLower(v))
	switch s {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return true
	}
}
