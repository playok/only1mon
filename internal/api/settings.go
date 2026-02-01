package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/playok/only1mon/internal/collector"
	"github.com/playok/only1mon/internal/store"
)

type settingsAPI struct {
	store     *store.Store
	scheduler *collector.Scheduler
}

func (a *settingsAPI) list(w http.ResponseWriter, r *http.Request) {
	settings, err := a.store.GetAllSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// Convert to map for easier consumption
	m := make(map[string]string)
	for _, s := range settings {
		m[s.Key] = s.Value
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *settingsAPI) update(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	for k, v := range body {
		if err := a.store.SetSetting(k, v); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	// Apply collect_interval change to running scheduler
	if v, ok := body["collect_interval"]; ok && a.scheduler != nil {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			a.scheduler.UpdateInterval(sec)
		}
	}

	// Apply top_process_count change to running scheduler
	if v, ok := body["top_process_count"]; ok && a.scheduler != nil {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			a.scheduler.UpdateTopN(n)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (a *settingsAPI) dbInfo(w http.ResponseWriter, r *http.Request) {
	dbPath := a.store.DBPath()
	info := map[string]interface{}{
		"path": dbPath,
		"size": int64(0),
	}

	// Main DB file
	if fi, err := os.Stat(dbPath); err == nil {
		info["size"] = fi.Size()
	}

	// WAL file
	if fi, err := os.Stat(dbPath + "-wal"); err == nil {
		info["wal_size"] = fi.Size()
	}

	writeJSON(w, http.StatusOK, info)
}

func (a *settingsAPI) dbPurge(w http.ResponseWriter, r *http.Request) {
	deleted, err := a.store.PurgeAllMetricSamples()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "purged",
		"deleted": deleted,
	})
}
