package config

import (
	"flag"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Listen   string `yaml:"listen"`
	DBPath   string `yaml:"database"`
	BasePath string `yaml:"base_path"`
	PidFile  string `yaml:"pid_file"`
	LogFile  string `yaml:"log_file"`

	// Runtime settings (managed via UI / DB, not in YAML)
	CollectInterval int `yaml:"-"`
	RetentionHours  int `yaml:"-"`

	// Parsed from command line (not YAML)
	ConfigPath string `yaml:"-"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Listen:          "127.0.0.1:9923",
		DBPath:          "only1mon.db",
		BasePath:        "/",
		PidFile:         "only1mon.pid",
		LogFile:         "only1mon.log",
		CollectInterval: 5,
		RetentionHours:  24,
		ConfigPath:      "config.yaml",
	}
}

// Load reads configuration with priority: defaults < config.yaml < env vars < flags.
// It expects os.Args to already have the subcommand stripped (if any).
func Load() *Config {
	cfg := DefaultConfig()

	// 1) Pre-scan for -config flag before parsing (so we know which file to read)
	configPath := cfg.ConfigPath
	for i, arg := range os.Args[1:] {
		if arg == "-config" || arg == "--config" {
			if i+1 < len(os.Args)-2 {
				configPath = os.Args[i+2]
			}
		} else if strings.HasPrefix(arg, "-config=") || strings.HasPrefix(arg, "--config=") {
			configPath = strings.SplitN(arg, "=", 2)[1]
		}
	}

	// 2) Load YAML config file
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Printf("[config] warning: failed to parse %s: %v", configPath, err)
		} else {
			log.Printf("[config] loaded %s", configPath)
		}
	}
	cfg.ConfigPath = configPath

	// 3) Environment variables override YAML
	if v := os.Getenv("ONLY1MON_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := os.Getenv("ONLY1MON_DB"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("ONLY1MON_BASE_PATH"); v != "" {
		cfg.BasePath = v
	}

	// 4) Flags override everything
	flag.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "Path to config.yaml")
	flag.StringVar(&cfg.Listen, "listen", cfg.Listen, "HTTP listen address (host:port)")
	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "SQLite database path")
	flag.StringVar(&cfg.BasePath, "base-path", cfg.BasePath, "Base URL path for reverse proxy")
	flag.StringVar(&cfg.PidFile, "pid-file", cfg.PidFile, "PID file path")
	flag.StringVar(&cfg.LogFile, "log-file", cfg.LogFile, "Log file path")
	flag.Parse()

	// Normalize base_path
	cfg.BasePath = normalizeBasePath(cfg.BasePath)

	return cfg
}

// normalizeBasePath ensures the base path starts with "/" and has no trailing "/".
// Returns "/" for empty or root paths.
func normalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimRight(p, "/")
	return p
}
