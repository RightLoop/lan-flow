package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "serve":
		if err := runServe(); err != nil {
			log.Fatal(err)
		}
	case "status":
		if err := printStatus(); err != nil {
			log.Fatal(err)
		}
	case "install-startup":
		if err := installStartup(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Lan Trans startup task installed.")
	case "uninstall-startup":
		if err := uninstallStartup(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Lan Trans startup task removed.")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		fmt.Fprintln(os.Stderr, "usage: lan-trans.exe [serve|status|install-startup|uninstall-startup]")
		os.Exit(2)
	}
}

func runServe() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	// Check for another running instance before proceeding.
	if err := checkLock(cfg); err != nil {
		return err
	}

	store, err := NewStore(cfg)
	if err != nil {
		return err
	}
	if err := store.CleanupExpired(); err != nil {
		log.Printf("cleanup failed: %v", err)
	}
	startCleanupLoop(store, cfg)

	return Serve(cfg, store)
}

func printStatus() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	runtime, err := ReadRuntime(cfg)
	if err != nil {
		fmt.Printf("Status: not running or runtime file unavailable (%v)\n", err)
		fmt.Printf("Data directory: %s\n", cfg.DataDir)
		return nil
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(runtime.Port)), 500*time.Millisecond)
	if err != nil {
		fmt.Printf("Status: not running; stale runtime file found\n")
		fmt.Printf("Last listen: %s:%d\n", runtime.ListenHost, runtime.Port)
		fmt.Printf("Data directory: %s\n", cfg.DataDir)
		return nil
	}
	_ = conn.Close()

	fmt.Printf("Status: running\n")
	fmt.Printf("Listen: %s:%d\n", runtime.ListenHost, runtime.Port)
	fmt.Printf("Data directory: %s\n", cfg.DataDir)
	for _, addr := range runtime.URLs {
		fmt.Printf("URL: %s\n", addr)
	}
	return nil
}
