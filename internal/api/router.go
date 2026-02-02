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

	// Prefix for direct access (empty when base_path is "/")
	bp := ""
	if basePath != "/" && basePath != "" {
		bp = basePath
	}

	// register adds a route and, if base_path is set, a prefixed duplicate.
	register := func(pattern string, handler func(http.ResponseWriter, *http.Request)) {
		mux.HandleFunc(pattern, handler)
		if bp != "" {
			method, path, ok := strings.Cut(pattern, " ")
			if ok {
				mux.HandleFunc(method+" "+bp+path, handler)
			} else {
				mux.HandleFunc(bp+pattern, handler)
			}
		}
	}

	// Collectors
	register("GET /api/v1/collectors", ca.list)
	register("PUT /api/v1/collectors/{id}/enable", ca.enable)
	register("PUT /api/v1/collectors/{id}/disable", ca.disable)
	register("PUT /api/v1/collectors/{id}/metrics/enable", ca.enableCollectorMetrics)
	register("PUT /api/v1/collectors/{id}/metrics/disable", ca.disableCollectorMetrics)

	// Metrics
	register("GET /api/v1/metrics/available", ma.available)
	register("GET /api/v1/metrics/query", ma.query)
	register("PUT /api/v1/metrics/state/{rest...}", ca.metricState)
	register("PUT /api/v1/metrics/ensure-enabled", ca.ensureMetricsEnabled)

	// Settings
	register("GET /api/v1/settings", sa.list)
	register("PUT /api/v1/settings", sa.update)
	register("GET /api/v1/settings/db-info", sa.dbInfo)
	register("DELETE /api/v1/settings/db-purge", sa.dbPurge)

	// Dashboard layouts
	register("GET /api/v1/dashboard/layouts", da.list)
	register("POST /api/v1/dashboard/layouts", da.create)
	register("GET /api/v1/dashboard/layouts/{id}", da.get)
	register("PUT /api/v1/dashboard/layouts/{id}", da.update)
	register("DELETE /api/v1/dashboard/layouts/{id}", da.delete)

	// Alerts
	register("GET /api/v1/alerts", aa.list)

	// Alert Rules
	register("GET /api/v1/alert-rules", aa.listRules)
	register("POST /api/v1/alert-rules", aa.createRule)
	register("PUT /api/v1/alert-rules/{id}", aa.updateRule)
	register("DELETE /api/v1/alert-rules/{id}", aa.deleteRule)

	// WebSocket
	register("GET /api/v1/ws", hub.HandleWS)

	// Static files (embedded) — inject base_path into index.html
	staticHandler := web.StaticHandler(basePath)
	mux.Handle("/", staticHandler)
	if bp != "" {
		mux.Handle(bp+"/", http.StripPrefix(bp, staticHandler))
	}

	return withMiddleware(mux)
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
