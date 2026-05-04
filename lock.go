package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const lockFilename = "lan-flow.lock"

func lockPath(cfg Config) string {
	return filepath.Join(cfg.DataDir, lockFilename)
}

// checkLock checks whether another instance is already running.
//   - No lock file → OK (return nil)
//   - Stale lock from crashed instance → clean it up → OK (return nil)
//   - Active lock from running instance → return error
func checkLock(cfg Config) error {
	path := lockPath(cfg)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	portStr := strings.TrimSpace(string(data))
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		_ = os.Remove(path)
		return nil
	}

	// Try connecting to the port recorded in the lock file.
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", portStr), 300*time.Millisecond)
	if err != nil {
		// Port is not reachable — stale lock from a previous crash.
		_ = os.Remove(path)
		log.Printf("removed stale lock from previous instance (port %d)", port)
		return nil
	}
	conn.Close()

	return fmt.Errorf(
		"cannot start: another Lan Trans instance is already running on port %d\n"+
			"  Lock file: %s\n"+
			"  Run 'lan-flow.exe status' to check the running instance",
		port, path)
}

func writeLock(cfg Config, port int) error {
	return os.WriteFile(lockPath(cfg), []byte(strconv.Itoa(port)), 0644)
}

func removeLock(cfg Config) {
	if err := os.Remove(lockPath(cfg)); err == nil {
		log.Println("lock file removed")
	}
}
