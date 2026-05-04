package main

import (
	"log"
	"time"
)

func startCleanupLoop(store *Store, cfg Config) {
	interval := time.Duration(cfg.CleanupIntervalMinutes) * time.Minute
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := store.CleanupExpired(); err != nil {
				log.Printf("cleanup failed: %v", err)
			}
		}
	}()
}

