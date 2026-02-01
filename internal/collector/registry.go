package collector

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/only1mon/only1mon/internal/model"
	"github.com/only1mon/only1mon/internal/store"
)

// Registry manages collector registration and enabled state.
type Registry struct {
	mu                sync.RWMutex
	collectors        map[string]Collector
	enabled           map[string]bool
	disabledMetrics   map[string]bool     // opt-out: only disabled metrics are tracked
	discoveredMetrics map[string][]string  // collector ID → actual metric names from system
	store             *store.Store
}

// NewRegistry creates a new collector registry and restores state from DB.
func NewRegistry(s *store.Store) *Registry {
	r := &Registry{
		collectors:        make(map[string]Collector),
		enabled:           make(map[string]bool),
		disabledMetrics:   make(map[string]bool),
		discoveredMetrics: make(map[string][]string),
		store:             s,
	}
	return r
}

// Register adds a collector to the registry.
func (r *Registry) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[c.ID()] = c
}

// RestoreState loads enabled states from the database.
func (r *Registry) RestoreState() error {
	states, err := r.store.GetAllCollectorStates()
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range states {
		r.enabled[s.CollectorID] = s.Enabled
	}
	return nil
}

// Enable enables a collector and saves state to DB.
func (r *Registry) Enable(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.collectors[id]; !ok {
		return ErrCollectorNotFound
	}
	r.enabled[id] = true
	return r.store.SetCollectorEnabled(id, true)
}

// Disable disables a collector and saves state to DB.
func (r *Registry) Disable(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.collectors[id]; !ok {
		return ErrCollectorNotFound
	}
	r.enabled[id] = false
	return r.store.SetCollectorEnabled(id, false)
}

// IsEnabled returns whether a collector is enabled.
func (r *Registry) IsEnabled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled[id]
}

// GetCollector returns a collector by ID.
func (r *Registry) GetCollector(id string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[id]
	return c, ok
}

// DiscoverMetrics runs every registered collector once to discover the actual
// metric names available on this system (e.g. per-core, per-disk, per-interface).
// Results are cached in-memory and used by ListCollectors / SetCollectorMetrics.
func (r *Registry) DiscoverMetrics(ctx context.Context) {
	r.mu.RLock()
	ids := make([]string, 0, len(r.collectors))
	for id := range r.collectors {
		ids = append(ids, id)
	}
	r.mu.RUnlock()

	for _, id := range ids {
		r.mu.RLock()
		c := r.collectors[id]
		r.mu.RUnlock()

		// Call Collect twice: the first call establishes baseline state for
		// collectors that use delta-based calculations (e.g. CPU times).
		// The second call then produces the full set of metrics.
		c.Collect(ctx)
		samples, err := c.Collect(ctx)
		if err != nil {
			log.Printf("[discover] collector %s error: %v", id, err)
			continue
		}

		seen := make(map[string]bool)
		var names []string
		for _, s := range samples {
			if !seen[s.MetricName] {
				seen[s.MetricName] = true
				names = append(names, s.MetricName)
			}
		}
		sort.Strings(names)

		r.mu.Lock()
		r.discoveredMetrics[id] = names
		r.mu.Unlock()

		log.Printf("[discover] %s: %d metrics found", id, len(names))
	}
}

// collectorMetrics returns discovered metric names for a collector,
// falling back to the declared MetricNames() if discovery hasn't run.
func (r *Registry) collectorMetrics(c Collector) []string {
	if discovered, ok := r.discoveredMetrics[c.ID()]; ok && len(discovered) > 0 {
		return discovered
	}
	return c.MetricNames()
}

// IsMetricEnabled returns whether a specific metric is enabled (opt-out model).
func (r *Registry) IsMetricEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return !r.disabledMetrics[name]
}

// EnsureMetricsEnabled enables multiple metrics at once (only those currently disabled).
// It also auto-enables the parent collector for any requested metrics.
func (r *Registry) EnsureMetricsEnabled(names []string) error {
	r.mu.Lock()

	// 1. Enable individual metrics (opt-out removal)
	var toEnable []string
	for _, name := range names {
		if r.disabledMetrics[name] {
			delete(r.disabledMetrics, name)
			toEnable = append(toEnable, name)
		}
	}

	// 2. Find and auto-enable parent collectors for the requested metrics
	collectorsToEnable := r.findCollectorsForMetrics(names)
	r.mu.Unlock()

	// Enable collectors that are currently disabled
	for _, cid := range collectorsToEnable {
		if err := r.Enable(cid); err != nil {
			log.Printf("[registry] failed to auto-enable collector %s: %v", cid, err)
		} else {
			log.Printf("[registry] auto-enabled collector %s for requested metrics", cid)
		}
	}

	if len(toEnable) == 0 {
		return nil
	}
	return r.store.SetMetricsBulkEnabled(toEnable, true)
}

// findCollectorsForMetrics returns IDs of disabled collectors that own any of the given metric names.
// Matches by exact metric name (discovered/declared) and also by first-segment prefix
// for dynamic metrics like "proc.top_cpu.0.pid" that aren't in the declared list.
// Must be called with r.mu held (at least RLock).
func (r *Registry) findCollectorsForMetrics(names []string) []string {
	requested := make(map[string]bool, len(names))
	requestedPrefixes := make(map[string]bool)
	for _, n := range names {
		requested[n] = true
		if idx := strings.IndexByte(n, '.'); idx > 0 {
			requestedPrefixes[n[:idx]] = true
		}
	}

	var result []string
	for id, c := range r.collectors {
		if r.enabled[id] {
			continue
		}

		// Check discovered or declared metrics for exact match or prefix match
		metrics := r.discoveredMetrics[id]
		if len(metrics) == 0 {
			metrics = c.MetricNames()
		}

		found := false
		for _, m := range metrics {
			if requested[m] {
				found = true
				break
			}
			// Prefix match: "proc.total_count" → prefix "proc"
			if idx := strings.IndexByte(m, '.'); idx > 0 {
				if requestedPrefixes[m[:idx]] {
					found = true
					break
				}
			}
		}
		if found {
			result = append(result, id)
		}
	}

	return result
}

// EnableMetric enables a specific metric and persists to DB.
func (r *Registry) EnableMetric(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.disabledMetrics, name)
	return r.store.SetMetricEnabled(name, true)
}

// DisableMetric disables a specific metric and persists to DB.
func (r *Registry) DisableMetric(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disabledMetrics[name] = true
	return r.store.SetMetricEnabled(name, false)
}

// SetCollectorMetrics enables or disables all metrics for a collector.
func (r *Registry) SetCollectorMetrics(collectorID string, enabled bool) error {
	r.mu.Lock()
	c, ok := r.collectors[collectorID]
	if !ok {
		r.mu.Unlock()
		return ErrCollectorNotFound
	}
	names := r.collectorMetrics(c)
	for _, name := range names {
		if enabled {
			delete(r.disabledMetrics, name)
		} else {
			r.disabledMetrics[name] = true
		}
	}
	r.mu.Unlock()
	return r.store.SetMetricsBulkEnabled(names, enabled)
}

// RestoreMetricStates loads disabled metric states from the database.
func (r *Registry) RestoreMetricStates() error {
	disabled, err := r.store.GetDisabledMetrics()
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for name := range disabled {
		r.disabledMetrics[name] = true
	}
	return nil
}

// ListCollectors returns info about all registered collectors.
func (r *Registry) ListCollectors() []model.CollectorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []model.CollectorInfo
	for _, c := range r.collectors {
		metrics := r.collectorMetrics(c)
		states := make([]model.MetricState, len(metrics))
		for i, m := range metrics {
			desc := LookupMetricDesc(m)
			states[i] = model.MetricState{
				Name:          m,
				Enabled:       !r.disabledMetrics[m],
				Description:   desc.Description,
				DescriptionKO: desc.DescriptionKO,
				Unit:          desc.Unit,
			}
		}
		result = append(result, model.CollectorInfo{
			ID:           c.ID(),
			Name:         c.Name(),
			Description:  c.Description(),
			Impact:       c.Impact(),
			Warning:      c.Warning(),
			Enabled:      r.enabled[c.ID()],
			Metrics:      metrics,
			MetricStates: states,
		})
	}
	return result
}

// EnabledCollectors returns all currently enabled collectors.
func (r *Registry) EnabledCollectors() []Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Collector
	for id, c := range r.collectors {
		if r.enabled[id] {
			result = append(result, c)
		}
	}
	return result
}

// HasAnyState returns true if any collector state has been saved to DB.
func (r *Registry) HasAnyState() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.enabled) > 0
}

// SetProcessTopN finds the process collector and sets topN.
func (r *Registry) SetProcessTopN(n int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.collectors {
		if pc, ok := c.(*processCollector); ok {
			pc.SetTopN(n)
			return
		}
	}
}

// errors
var ErrCollectorNotFound = &CollectorError{"collector not found"}

type CollectorError struct {
	msg string
}

func (e *CollectorError) Error() string { return e.msg }
