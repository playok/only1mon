package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/playok/only1mon/internal/collector"
	"github.com/playok/only1mon/internal/store"
)

type metricsAPI struct {
	store    *store.Store
	registry *collector.Registry
}

func (a *metricsAPI) query(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name parameter required"})
		return
	}

	now := time.Now().Unix()

	fromStr := r.URL.Query().Get("from")
	from := now - 3600 // default: last hour
	if fromStr != "" {
		if v, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
			from = v
		}
	}

	toStr := r.URL.Query().Get("to")
	to := now
	if toStr != "" {
		if v, err := strconv.ParseInt(toStr, 10, 64); err == nil {
			to = v
		}
	}

	stepStr := r.URL.Query().Get("step")
	step := 0
	if stepStr != "" {
		if v, err := strconv.Atoi(stepStr); err == nil {
			step = v
		}
	}

	// Support comma-separated names
	names := strings.Split(name, ",")
	if len(names) == 1 {
		samples, err := a.store.QueryMetrics(names[0], from, to, step)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, samples)
	} else {
		samples, err := a.store.QueryMultiMetrics(names, from, to, step)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, samples)
	}
}

// metricInfo represents a single metric with its description.
type metricInfo struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	DescriptionKO string `json:"description_ko,omitempty"`
	Unit          string `json:"unit,omitempty"`
}

// metricGroup is a hierarchical group of metrics for the available endpoint.
type metricGroup struct {
	Label    string         `json:"label"`
	Metrics  []metricInfo   `json:"metrics,omitempty"`
	Children []*metricGroup `json:"children,omitempty"`
}

// available returns all metrics grouped by collector → sub-group.
// It merges collector-declared patterns with actually-collected metric names.
func (a *metricsAPI) available(w http.ResponseWriter, r *http.Request) {
	// 1. Get actually collected distinct metrics from DB
	metas, err := a.store.GetDistinctMetrics()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Build set of real metric names per collector
	realByCollector := map[string][]string{}
	for _, m := range metas {
		realByCollector[m.Collector] = append(realByCollector[m.Collector], m.MetricName)
	}

	// 2. Get declared patterns from each collector in the registry
	collectors := a.registry.ListCollectors()
	sort.Slice(collectors, func(i, j int) bool {
		return collectors[i].ID < collectors[j].ID
	})

	var groups []*metricGroup
	for _, c := range collectors {
		// Merge: use real metrics if available, fall back to declared patterns
		metrics := realByCollector[c.ID]
		if len(metrics) == 0 {
			metrics = c.Metrics
		}
		sort.Strings(metrics)

		// Build metricInfo with descriptions
		toInfo := func(names []string) []metricInfo {
			infos := make([]metricInfo, len(names))
			for i, name := range names {
				desc := collector.LookupMetricDesc(name)
				infos[i] = metricInfo{
					Name:          name,
					Description:   desc.Description,
					DescriptionKO: desc.DescriptionKO,
					Unit:          desc.Unit,
				}
			}
			return infos
		}

		// Sub-group by the second segment of the metric name
		// e.g. cpu.total.user → sub-group "total", cpu.load.1 → sub-group "load"
		subGroups := map[string][]string{}
		var subOrder []string
		for _, m := range metrics {
			parts := strings.SplitN(m, ".", 3)
			sub := "(root)"
			if len(parts) >= 2 {
				sub = parts[1]
			}
			if _, ok := subGroups[sub]; !ok {
				subOrder = append(subOrder, sub)
			}
			subGroups[sub] = append(subGroups[sub], m)
		}

		g := &metricGroup{
			Label: c.Name + " (" + c.ID + ")",
		}

		if len(subOrder) == 1 {
			g.Metrics = toInfo(subGroups[subOrder[0]])
		} else {
			for _, sub := range subOrder {
				g.Children = append(g.Children, &metricGroup{
					Label:   sub,
					Metrics: toInfo(subGroups[sub]),
				})
			}
		}

		groups = append(groups, g)
	}

	writeJSON(w, http.StatusOK, groups)
}
