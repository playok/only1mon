package model

// MetricSample represents a single metric data point.
type MetricSample struct {
	ID         int64   `json:"id,omitempty"`
	Timestamp  int64   `json:"timestamp"`
	Collector  string  `json:"collector"`
	MetricName string  `json:"metric_name"`
	Value      float64 `json:"value"`
	Labels     string  `json:"labels,omitempty"`
}

// MetricMeta describes a metric's metadata.
type MetricMeta struct {
	MetricName string `json:"metric_name"`
	Collector  string `json:"collector"`
}
