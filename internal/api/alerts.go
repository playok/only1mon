package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/only1mon/only1mon/internal/collector"
	"github.com/only1mon/only1mon/internal/model"
	"github.com/only1mon/only1mon/internal/store"
)

type alertsAPI struct {
	alertEngine *collector.AlertEngine
	store       *store.Store
}

func (a *alertsAPI) list(w http.ResponseWriter, r *http.Request) {
	alerts := a.alertEngine.ActiveAlerts()
	if alerts == nil {
		alerts = []model.Alert{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// --- Alert Rules CRUD ---

func (a *alertsAPI) listRules(w http.ResponseWriter, r *http.Request) {
	rules, err := a.store.ListAlertRules()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if rules == nil {
		rules = []model.AlertRule{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (a *alertsAPI) createRule(w http.ResponseWriter, r *http.Request) {
	var rule model.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	id, err := a.store.CreateAlertRule(&rule)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	rule.ID = id
	a.alertEngine.ReloadRules(a.store)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (a *alertsAPI) updateRule(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	var rule model.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	rule.ID = id
	if err := a.store.UpdateAlertRule(&rule); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	a.alertEngine.ReloadRules(a.store)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

func (a *alertsAPI) deleteRule(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	if err := a.store.DeleteAlertRule(id); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	a.alertEngine.ReloadRules(a.store)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
