package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type RuntimeInfo struct {
	ListenHost string    `json:"listenHost"`
	Port       int       `json:"port"`
	URLs       []string  `json:"urls"`
	StartedAt  time.Time `json:"startedAt"`
}

func WriteRuntime(cfg Config, info RuntimeInfo) error {
	return writeJSONAtomic(runtimePath(cfg), info)
}

func ReadRuntime(cfg Config) (RuntimeInfo, error) {
	var info RuntimeInfo
	data, err := os.ReadFile(runtimePath(cfg))
	if err != nil {
		return info, err
	}
	return info, json.Unmarshal(data, &info)
}

func runtimePath(cfg Config) string {
	return filepath.Join(cfg.DataDir, "runtime.json")
}

