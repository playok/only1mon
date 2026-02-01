package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/playok/only1mon/internal/model"
	"github.com/playok/only1mon/internal/store"
)

type dashboardAPI struct {
	store *store.Store
}

func (a *dashboardAPI) list(w http.ResponseWriter, r *http.Request) {
	layouts, err := a.store.ListDashboardLayouts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if layouts == nil {
		layouts = []model.DashboardLayout{}
	}
	writeJSON(w, http.StatusOK, layouts)
}

func (a *dashboardAPI) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	layout, err := a.store.GetDashboardLayout(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if layout == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, layout)
}

func (a *dashboardAPI) create(w http.ResponseWriter, r *http.Request) {
	var dl model.DashboardLayout
	if err := json.NewDecoder(r.Body).Decode(&dl); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	id, err := a.store.CreateDashboardLayout(&dl)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	dl.ID = id
	writeJSON(w, http.StatusCreated, dl)
}

func (a *dashboardAPI) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var dl model.DashboardLayout
	if err := json.NewDecoder(r.Body).Decode(&dl); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	dl.ID = id
	if err := a.store.UpdateDashboardLayout(&dl); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, dl)
}

func (a *dashboardAPI) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := a.store.DeleteDashboardLayout(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
