package model

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
	SeverityInfo     AlertSeverity = "info"
)

// Alert represents a performance event/alert generated from metric analysis.
type Alert struct {
	ID         string        `json:"id"`
	Timestamp  int64         `json:"timestamp"`
	Severity   AlertSeverity `json:"severity"`
	Metric     string        `json:"metric"`
	Value      float64       `json:"value"`
	Threshold  float64       `json:"threshold"`
	MessageEN  string        `json:"message_en"`
	MessageKO  string        `json:"message_ko"`
}

// AlertRule defines a user-configurable rule that triggers alerts.
type AlertRule struct {
	ID            int64         `json:"id"`
	MetricPattern string        `json:"metric_pattern"`
	Operator      string        `json:"operator"`
	Threshold     float64       `json:"threshold"`
	Severity      AlertSeverity `json:"severity"`
	MessageEN     string        `json:"message_en"`
	MessageKO     string        `json:"message_ko"`
	Enabled       bool          `json:"enabled"`
}
