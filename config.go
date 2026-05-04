package main

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
)

const appName = "LanFlow"

type Config struct {
	ListenHost             string   `json:"listenHost"`
	PreferredPort          int      `json:"preferredPort"`
	PortFallbackEnd        int      `json:"portFallbackEnd"`
	RetentionHours         int      `json:"retentionHours"`
	CleanupIntervalMinutes int      `json:"cleanupIntervalMinutes"`
	MaxUploadMB            int64    `json:"maxUploadMB"`
	AllowCIDRs             []string `json:"allowCIDRs"`
	DataDir                string   `json:"dataDir"`
}

func DefaultConfig() Config {
	dir := defaultDataDir()
	return Config{
		ListenHost:             "0.0.0.0",
		PreferredPort:          8787,
		PortFallbackEnd:        8807,
		RetentionHours:         24,
		CleanupIntervalMinutes: 10,
		MaxUploadMB:            2048,
		AllowCIDRs: []string{
			"127.0.0.1/32",
			"::1/128",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		},
		DataDir: dir,
	}
}

func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return cfg, err
	}

	path := filepath.Join(cfg.DataDir, "config.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, writeJSONAtomic(path, cfg)
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	normalizeConfig(&cfg)
	return cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.ListenHost == "" {
		cfg.ListenHost = "0.0.0.0"
	}
	if cfg.PreferredPort == 0 {
		cfg.PreferredPort = 8787
	}
	if cfg.PortFallbackEnd < cfg.PreferredPort {
		cfg.PortFallbackEnd = cfg.PreferredPort
	}
	if cfg.RetentionHours <= 0 {
		cfg.RetentionHours = 24
	}
	if cfg.CleanupIntervalMinutes <= 0 {
		cfg.CleanupIntervalMinutes = 10
	}
	if cfg.MaxUploadMB <= 0 {
		cfg.MaxUploadMB = 2048
	}
	if len(cfg.AllowCIDRs) == 0 {
		cfg.AllowCIDRs = DefaultConfig().AllowCIDRs
	}
	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir()
	}
}

func defaultDataDir() string {
	if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
		return filepath.Join(dir, appName)
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, appName)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "."+appName)
	}
	return appName
}

func parseAllowedCIDRs(cidrs []string) ([]*net.IPNet, error) {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		out = append(out, network)
	}
	return out, nil
}

