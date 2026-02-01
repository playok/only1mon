package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

// StaticHandler returns an http.Handler that serves the embedded static files.
// It injects basePath into index.html so the frontend knows the URL prefix.
func StaticHandler(basePath string) http.Handler {
	sub, _ := fs.Sub(staticFS, "static")
	fileServer := http.FileServerFS(sub)

	// Read index.html once at startup for injection
	indexBytes, _ := fs.ReadFile(sub, "index.html")
	indexHTML := string(indexBytes)

	// Inject base_path script tag right after <head>
	injected := strings.Replace(indexHTML,
		"<head>",
		"<head>\n<script>window.__BASE_PATH='"+basePath+"';</script>",
		1,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve injected index.html for root or index.html requests
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(injected))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
