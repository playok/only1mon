package model

// DashboardLayout represents a saved dashboard layout.
type DashboardLayout struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name"`
	Layout  string `json:"layout"`
	Updated int64  `json:"updated"`
}
