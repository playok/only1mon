package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/only1mon/only1mon/internal/collector"
	"github.com/only1mon/only1mon/internal/store"
	"github.com/only1mon/only1mon/web"
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

	// Static files (embedded) — inject base_path into index.html
	mux.Handle("/", web.StaticHandler(basePath))

	var handler http.Handler = mux

	// If base_path is set, strip the prefix so internal routing works unchanged
	if basePath != "/" && basePath != "" {
		inner := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strip base path prefix from the URL
			if strings.HasPrefix(r.URL.Path, basePath) {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, basePath)
				if r.URL.Path == "" {
					r.URL.Path = "/"
				}
				r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, basePath)
			}
			inner.ServeHTTP(w, r)
		})
	}

	return withMiddleware(handler)
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

		next.ServeHTTP(w, r)

		log.Printf("[http] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
