package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"time"
)

func Serve(cfg Config, store *Store) error {
	listener, port, err := listenWithFallback(cfg)
	if err != nil {
		return err
	}

	// Write lock file to prevent duplicate instances.
	if err := writeLock(cfg, port); err != nil {
		log.Printf("warning: failed to write lock file: %v", err)
	}
	defer removeLock(cfg)

	urls := accessURLs(port)
	info := RuntimeInfo{
		ListenHost: cfg.ListenHost,
		Port:       port,
		URLs:       urls,
		StartedAt:  time.Now(),
	}
	if err := WriteRuntime(cfg, info); err != nil {
		return err
	}

	handler, err := accessMiddleware(cfg.AllowCIDRs, routes(cfg, store, info))
	if err != nil {
		return err
	}

	log.Printf("Lan Trans listening on %s:%d", cfg.ListenHost, port)
	for _, url := range urls {
		log.Printf("URL: %s", url)
	}

	// Use an explicit http.Server so we can shut down gracefully on Ctrl+C.
	srv := &http.Server{Handler: handler}
	go handleShutdownSignal(srv)

	err = srv.Serve(listener)
	if err == http.ErrServerClosed {
		return nil // graceful shutdown
	}
	return err
}

func handleShutdownSignal(srv *http.Server) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func listenWithFallback(cfg Config) (net.Listener, int, error) {
	var lastErr error
	for port := cfg.PreferredPort; port <= cfg.PortFallbackEnd; port++ {
		addr := net.JoinHostPort(cfg.ListenHost, strconv.Itoa(port))
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			if port != cfg.PreferredPort {
				log.Printf("WARNING: preferred port %d is unavailable (possibly another instance or app is using it), falling back to port %d", cfg.PreferredPort, port)
			}
			return ln, port, nil
		}
		lastErr = err
	}
	return nil, 0, fmt.Errorf("no available port in %d-%d: %w", cfg.PreferredPort, cfg.PortFallbackEnd, lastErr)
}

func routes(cfg Config, store *Store, runtime RuntimeInfo) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ok":             true,
			"port":           runtime.Port,
			"urls":           runtime.URLs,
			"retentionHours": cfg.RetentionHours,
			"maxUploadMB":    cfg.MaxUploadMB,
		})
	})
	mux.HandleFunc("GET /api/items", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, store.Items())
	})
	mux.HandleFunc("GET /api/messages", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, store.Items().Messages)
	})
	mux.HandleFunc("POST /api/messages", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128*1024)).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		item, err := store.AddMessage(req.Text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, item)
	})
	mux.HandleFunc("DELETE /api/messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !store.DeleteMessage(r.PathValue("id")) {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/files", func(w http.ResponseWriter, r *http.Request) {
		maxBytes := cfg.MaxUploadMB * 1024 * 1024
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, "multipart form is required", http.StatusBadRequest)
			return
		}
		part, err := nextFilePart(reader)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer part.Close()
		item, err := store.AddFile(part.FileName(), part)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, item)
	})
	mux.HandleFunc("GET /api/files/{id}", func(w http.ResponseWriter, r *http.Request) {
		item, filePath, ok := store.FilePath(r.PathValue("id"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Disposition", contentDisposition(item.Name))
		if ctype := mime.TypeByExtension(path.Ext(item.Name)); ctype != "" {
			w.Header().Set("Content-Type", ctype)
		}
		http.ServeFile(w, r, filePath)
	})
	mux.HandleFunc("DELETE /api/files/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !store.DeleteFile(r.PathValue("id")) {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	static, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(static)))
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func nextFilePart(reader *multipart.Reader) (*multipart.Part, error) {
	for {
		part, err := reader.NextPart()
		if err != nil {
			return nil, errors.New("file is required")
		}
		if part.FormName() == "file" && part.FileName() != "" {
			return part, nil
		}
		_ = part.Close()
	}
}

func contentDisposition(name string) string {
	escaped := strings.ReplaceAll(name, `"`, "'")
	return fmt.Sprintf(`attachment; filename="%s"`, escaped)
}

func accessURLs(port int) []string {
	ips := localIPv4s()
	urls := make([]string, 0, len(ips)+1)
	for _, ip := range ips {
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, port))
	}
	urls = append(urls, fmt.Sprintf("http://127.0.0.1:%d", port))
	return urls
}
