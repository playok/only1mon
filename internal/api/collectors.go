package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/only1mon/only1mon/internal/collector"
)

type collectorsAPI struct {
	registry *collector.Registry
}

func (a *collectorsAPI) list(w http.ResponseWriter, r *http.Request) {
	infos := a.registry.ListCollectors()
	writeJSON(w, http.StatusOK, infos)
}

func (a *collectorsAPI) enable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.registry.Enable(id); err != nil {
		if err == collector.ErrCollectorNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "collector not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

func (a *collectorsAPI) disable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.registry.Disable(id); err != nil {
		if err == collector.ErrCollectorNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "collector not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

// metricState handles PUT /api/v1/metrics/state/{rest...}
// where rest is "<metric.name>/enable" or "<metric.name>/disable".
func (a *collectorsAPI) metricState(w http.ResponseWriter, r *http.Request) {
	rest := r.PathValue("rest") // e.g. "cpu.total.user/enable"

	var name, action string
	if i := strings.LastIndex(rest, "/"); i >= 0 {
		name = rest[:i]
		action = rest[i+1:]
	}
	if name == "" || (action != "enable" && action != "disable") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected /metrics/state/{name}/enable or /disable"})
		return
	}

	var err error
	if action == "enable" {
		err = a.registry.EnableMetric(name)
	} else {
		err = a.registry.DisableMetric(name)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": action + "d"})
}

func (a *collectorsAPI) enableCollectorMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.registry.SetCollectorMetrics(id, true); err != nil {
		if err == collector.ErrCollectorNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "collector not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

func (a *collectorsAPI) disableCollectorMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.registry.SetCollectorMetrics(id, false); err != nil {
		if err == collector.ErrCollectorNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "collector not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

func (a *collectorsAPI) ensureMetricsEnabled(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Metrics []string `json:"metrics"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := a.registry.EnsureMetricsEnabled(body.Metrics); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
