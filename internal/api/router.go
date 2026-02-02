package api

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/playok/only1mon/internal/collector"
	"github.com/playok/only1mon/internal/store"
	"github.com/playok/only1mon/web"
)

// NewRouter creates the HTTP router with all API routes.
func NewRouter(registry *collector.Registry, db *store.Store, hub *Hub, alertEngine *collector.AlertEngine, scheduler *collector.Scheduler, basePath string) http.Handler {
	mux := http.NewServeMux()

	ca := &collectorsAPI{registry: registry}
	ma := &metricsAPI{store: db, registry: registry}
	sa := &settingsAPI{store: db, scheduler: scheduler}
	da := &dashboardAPI{store: db}
	aa := &alertsAPI{alertEngine: alertEngine, store: db}

	// Collectors
	mux.HandleFunc("GET /api/v1/collectors", ca.list)
	mux.HandleFunc("PUT /api/v1/collectors/{id}/enable", ca.enable)
	mux.HandleFunc("PUT /api/v1/collectors/{id}/disable", ca.disable)
	mux.HandleFunc("PUT /api/v1/collectors/{id}/metrics/enable", ca.enableCollectorMetrics)
	mux.HandleFunc("PUT /api/v1/collectors/{id}/metrics/disable", ca.disableCollectorMetrics)

	// Metrics
	mux.HandleFunc("GET /api/v1/metrics/available", ma.available)
	mux.HandleFunc("GET /api/v1/metrics/query", ma.query)
	mux.HandleFunc("PUT /api/v1/metrics/state/{rest...}", ca.metricState)
	mux.HandleFunc("PUT /api/v1/metrics/ensure-enabled", ca.ensureMetricsEnabled)

	// Settings
	mux.HandleFunc("GET /api/v1/settings", sa.list)
	mux.HandleFunc("PUT /api/v1/settings", sa.update)
	mux.HandleFunc("GET /api/v1/settings/db-info", sa.dbInfo)
	mux.HandleFunc("DELETE /api/v1/settings/db-purge", sa.dbPurge)

	// Dashboard layouts
	mux.HandleFunc("GET /api/v1/dashboard/layouts", da.list)
	mux.HandleFunc("POST /api/v1/dashboard/layouts", da.create)
	mux.HandleFunc("GET /api/v1/dashboard/layouts/{id}", da.get)
	mux.HandleFunc("PUT /api/v1/dashboard/layouts/{id}", da.update)
	mux.HandleFunc("DELETE /api/v1/dashboard/layouts/{id}", da.delete)

	// Alerts
	mux.HandleFunc("GET /api/v1/alerts", aa.list)

	// Alert Rules
	mux.HandleFunc("GET /api/v1/alert-rules", aa.listRules)
	mux.HandleFunc("POST /api/v1/alert-rules", aa.createRule)
	mux.HandleFunc("PUT /api/v1/alert-rules/{id}", aa.updateRule)
	mux.HandleFunc("DELETE /api/v1/alert-rules/{id}", aa.deleteRule)

	// WebSocket
	mux.HandleFunc("GET /api/v1/ws", hub.HandleWS)

	// Catch-all for unmatched API routes — return JSON 404 instead of file server
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found", "path": r.URL.Path})
	})

	// Static files (embedded) — inject base_path into index.html
	mux.Handle("/", web.StaticHandler(basePath))

	var handler http.Handler = mux

	// If base_path is set, strip the prefix so internal routing works unchanged
	if basePath != "/" && basePath != "" {
		inner := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strip base path prefix from the URL
			if strings.HasPrefix(r.URL.Path, basePath) {
				p := strings.TrimPrefix(r.URL.Path, basePath)
				if p == "" {
					p = "/"
				}
				// Create a shallow copy so the original request is not mutated
				r2 := r.Clone(r.Context())
				r2.URL.Path = p
				r2.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, basePath)
				inner.ServeHTTP(w, r2)
				return
			}
			inner.ServeHTTP(w, r)
		})
	}

	return withMiddleware(handler)
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Hijack forwards to the underlying ResponseWriter for WebSocket upgrade support.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
}

// Flush forwards to the underlying ResponseWriter for streaming support.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter (Go 1.20+ convention).
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Recovery
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[http] panic: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		// CORS for local development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		log.Printf("[http] %d %s %s %s", sw.status, r.Method, r.RequestURI, time.Since(start))
	})
}
