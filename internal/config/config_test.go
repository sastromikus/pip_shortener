package config

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func withFreshFlags(t *testing.T, args []string, fn func()) {
	t.Helper()

	origArgs := os.Args
	origCmd := flag.CommandLine

	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = args

	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origCmd
	})

	fn()
}

func writeTempConfig(t *testing.T, dir string, content string) string {
	t.Helper()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestParse_ConfigFileLowPriority(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeTempConfig(t, dir, `{
  "server_address": "localhost:9999",
  "base_url": "http://example",
  "file_storage_path": "fromfile.json",
  "database_dsn": "dsn_from_file",
  "enable_https": true,
  "trusted_subnet": "127.0.0.0/8"
}`)

	withFreshFlags(t, []string{"cmd", "-c", cfgPath}, func() {
		cfg := Parse()
		if cfg.ServerAddr != "localhost:9999" {
			t.Fatalf("ServerAddr: want %q, got %q", "localhost:9999", cfg.ServerAddr)
		}
		if cfg.BaseURL != "http://example" {
			t.Fatalf("BaseURL: want %q, got %q", "http://example", cfg.BaseURL)
		}
		if cfg.FileStoragePath != "fromfile.json" {
			t.Fatalf("FileStoragePath: want %q, got %q", "fromfile.json", cfg.FileStoragePath)
		}
		if cfg.DatabaseDSN != "dsn_from_file" {
			t.Fatalf("DatabaseDSN: want %q, got %q", "dsn_from_file", cfg.DatabaseDSN)
		}
		if cfg.EnableHTTPS != true {
			t.Fatalf("EnableHTTPS: want true, got %v", cfg.EnableHTTPS)
		}
		if cfg.TrustedSubnet != "127.0.0.0/8" {
			t.Fatalf("TrustedSubnet: want %q, got %q", "127.0.0.0/8", cfg.TrustedSubnet)
		}
	})
}

func TestParse_EnvOverridesFlagsAndFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeTempConfig(t, dir, `{
  "server_address": "localhost:1111",
  "base_url": "http://file",
  "trusted_subnet": "10.0.0.0/8",
  "enable_https": false
}`)

	t.Setenv(envConfigPath, cfgPath)
	t.Setenv(envServerAddr, "localhost:3333")
	t.Setenv(envEnableHTTPS, "on")
	t.Setenv(envTrustedSubnet, "127.0.0.0/8")

	withFreshFlags(t, []string{"cmd", "-c", "ignored.json", "-a", "localhost:2222", "-t", "192.168.0.0/16"}, func() {
		cfg := Parse()
		if cfg.ServerAddr != "localhost:3333" {
			t.Fatalf("ServerAddr: want %q, got %q", "localhost:3333", cfg.ServerAddr)
		}
		if cfg.EnableHTTPS != true {
			t.Fatalf("EnableHTTPS: want true, got %v", cfg.EnableHTTPS)
		}
		if cfg.TrustedSubnet != "127.0.0.0/8" {
			t.Fatalf("TrustedSubnet: want %q, got %q", "127.0.0.0/8", cfg.TrustedSubnet)
		}
	})
}

func TestParse_EnableHTTPS_ParseEnvBool(t *testing.T) {
	t.Setenv(envEnableHTTPS, "off")
	withFreshFlags(t, []string{"cmd"}, func() {
		cfg := Parse()
		if cfg.EnableHTTPS {
			t.Fatalf("EnableHTTPS: want false, got true")
		}
	})

	t.Setenv(envEnableHTTPS, "true")
	withFreshFlags(t, []string{"cmd"}, func() {
		cfg := Parse()
		if !cfg.EnableHTTPS {
			t.Fatalf("EnableHTTPS: want true, got false")
		}
	})
}