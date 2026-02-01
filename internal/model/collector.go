package model

// CollectorState represents the enabled/disabled state and config of a collector.
type CollectorState struct {
	CollectorID string `json:"collector_id"`
	Enabled     bool   `json:"enabled"`
	ConfigJSON  string `json:"config_json,omitempty"`
}

// ImpactLevel describes the system load impact of a collector.
type ImpactLevel string

const (
	ImpactNone    ImpactLevel = "none"
	ImpactLow     ImpactLevel = "low"
	ImpactMedium  ImpactLevel = "medium"
	ImpactHigh    ImpactLevel = "high"
)

// MetricState describes the enabled state and metadata of a single metric.
type MetricState struct {
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	Description   string `json:"description,omitempty"`
	DescriptionKO string `json:"description_ko,omitempty"`
	Unit          string `json:"unit,omitempty"`
}

// CollectorInfo describes a collector for the API.
type CollectorInfo struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Impact       ImpactLevel   `json:"impact"`
	Warning      string        `json:"warning,omitempty"`
	Enabled      bool          `json:"enabled"`
	Metrics      []string      `json:"metrics"`
	MetricStates []MetricState `json:"metric_states,omitempty"`
}
