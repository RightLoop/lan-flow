package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed web/*
var webFS embed.FS

func webHandler() http.Handler {
	static, _ := fs.Sub(webFS, "web")
	files, _ := fs.ReadDir(static, ".")
	etags := make(map[string]string, len(files))
	for _, file := range files {
		if data, err := fs.ReadFile(static, file.Name()); err == nil {
			etag := fmt.Sprintf(`"%x"`, sha256.Sum256(data))
			etags["/"+file.Name()] = etag
			if file.Name() == "index.html" {
				etags["/"] = etag
			}
		}
	}

	fileServer := http.FileServer(http.FS(static))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := etags[r.URL.Path]; etag != "" {
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})
}
