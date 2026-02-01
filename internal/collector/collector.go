package collector

import (
	"context"

	"github.com/only1mon/only1mon/internal/model"
)

// Collector defines the interface for all metric collectors.
type Collector interface {
	// ID returns the unique identifier for this collector.
	ID() string
	// Name returns a human-readable name.
	Name() string
	// Description returns a description of what this collector does.
	Description() string
	// Impact returns the system load impact level.
	Impact() model.ImpactLevel
	// Warning returns an optional warning about using this collector.
	Warning() string
	// MetricNames returns the list of metric names this collector produces.
	MetricNames() []string
	// Collect gathers metrics and returns samples.
	Collect(ctx context.Context) ([]model.MetricSample, error)
}
