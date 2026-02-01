package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/only1mon/only1mon/internal/model"
	_ "modernc.org/sqlite"
)

// Store provides database operations.
type Store struct {
	db     *sql.DB
	dbPath string
}

// New opens (or creates) the SQLite database and runs migrations.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite single-writer
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}
	return &Store{db: db, dbPath: dbPath}, nil
}

// DBPath returns the database file path.
func (s *Store) DBPath() string { return s.dbPath }

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// InsertSamples batch-inserts metric samples.
func (s *Store) InsertSamples(samples []model.MetricSample) error {
	if len(samples) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO metric_samples (timestamp, collector, metric_name, value, labels) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, m := range samples {
		if _, err := stmt.Exec(m.Timestamp, m.Collector, m.MetricName, m.Value, m.Labels); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// QueryMetrics retrieves metric samples with optional downsampling.
// step is in seconds; if step > 0, data is averaged per step.
func (s *Store) QueryMetrics(name string, from, to int64, step int) ([]model.MetricSample, error) {
	var rows *sql.Rows
	var err error

	if step > 0 {
		rows, err = s.db.Query(`
			SELECT 0, (timestamp / ? * ?) as ts, collector, metric_name, AVG(value), labels
			FROM metric_samples
			WHERE metric_name = ? AND timestamp >= ? AND timestamp <= ?
			GROUP BY ts, labels
			ORDER BY ts`,
			step, step, name, from, to)
	} else {
		rows, err = s.db.Query(`
			SELECT id, timestamp, collector, metric_name, value, labels
			FROM metric_samples
			WHERE metric_name = ? AND timestamp >= ? AND timestamp <= ?
			ORDER BY timestamp`,
			name, from, to)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.MetricSample
	for rows.Next() {
		var m model.MetricSample
		if err := rows.Scan(&m.ID, &m.Timestamp, &m.Collector, &m.MetricName, &m.Value, &m.Labels); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryMultiMetrics retrieves samples for multiple metric names.
func (s *Store) QueryMultiMetrics(names []string, from, to int64, step int) ([]model.MetricSample, error) {
	if len(names) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(names))
	args := make([]interface{}, 0, len(names)+2)
	for i, n := range names {
		placeholders[i] = "?"
		args = append(args, n)
	}
	args = append(args, from, to)

	var query string
	if step > 0 {
		query = fmt.Sprintf(`
			SELECT 0, (timestamp / %d * %d) as ts, collector, metric_name, AVG(value), labels
			FROM metric_samples
			WHERE metric_name IN (%s) AND timestamp >= ? AND timestamp <= ?
			GROUP BY metric_name, ts, labels
			ORDER BY ts`, step, step, strings.Join(placeholders, ","))
	} else {
		query = fmt.Sprintf(`
			SELECT id, timestamp, collector, metric_name, value, labels
			FROM metric_samples
			WHERE metric_name IN (%s) AND timestamp >= ? AND timestamp <= ?
			ORDER BY timestamp`, strings.Join(placeholders, ","))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.MetricSample
	for rows.Next() {
		var m model.MetricSample
		if err := rows.Scan(&m.ID, &m.Timestamp, &m.Collector, &m.MetricName, &m.Value, &m.Labels); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// PurgeOlderThan removes samples older than the given duration.
func (s *Store) PurgeOlderThan(hours int) (int64, error) {
	cutoff := time.Now().Unix() - int64(hours*3600)
	res, err := s.db.Exec("DELETE FROM metric_samples WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PurgeAllMetricSamples deletes all metric sample data and reclaims disk space.
func (s *Store) PurgeAllMetricSamples() (int64, error) {
	res, err := s.db.Exec("DELETE FROM metric_samples")
	if err != nil {
		return 0, err
	}
	// Reclaim disk space
	s.db.Exec("VACUUM")
	return res.RowsAffected()
}

// GetDistinctMetrics returns all distinct (collector, metric_name) pairs.
func (s *Store) GetDistinctMetrics() ([]model.MetricMeta, error) {
	rows, err := s.db.Query("SELECT DISTINCT collector, metric_name FROM metric_samples ORDER BY collector, metric_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.MetricMeta
	for rows.Next() {
		var m model.MetricMeta
		if err := rows.Scan(&m.Collector, &m.MetricName); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// --- Collector State ---

// GetCollectorState returns the state for a collector.
func (s *Store) GetCollectorState(id string) (*model.CollectorState, error) {
	row := s.db.QueryRow("SELECT collector_id, enabled, config_json FROM collector_state WHERE collector_id = ?", id)
	var cs model.CollectorState
	var enabled int
	if err := row.Scan(&cs.CollectorID, &enabled, &cs.ConfigJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	cs.Enabled = enabled != 0
	return &cs, nil
}

// SetCollectorEnabled sets the enabled state of a collector.
func (s *Store) SetCollectorEnabled(id string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO collector_state (collector_id, enabled) VALUES (?, ?)
		ON CONFLICT(collector_id) DO UPDATE SET enabled = excluded.enabled`,
		id, enabledInt)
	return err
}

// GetAllCollectorStates returns all saved collector states.
func (s *Store) GetAllCollectorStates() ([]model.CollectorState, error) {
	rows, err := s.db.Query("SELECT collector_id, enabled, config_json FROM collector_state")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.CollectorState
	for rows.Next() {
		var cs model.CollectorState
		var enabled int
		if err := rows.Scan(&cs.CollectorID, &enabled, &cs.ConfigJSON); err != nil {
			return nil, err
		}
		cs.Enabled = enabled != 0
		result = append(result, cs)
	}
	return result, rows.Err()
}

// --- Settings ---

// GetSetting returns a setting value.
func (s *Store) GetSetting(key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetSetting upserts a setting.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// GetAllSettings returns all settings.
func (s *Store) GetAllSettings() ([]model.Setting, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.Setting
	for rows.Next() {
		var st model.Setting
		if err := rows.Scan(&st.Key, &st.Value); err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

// --- Metric State ---

// GetDisabledMetrics returns a set of metric names that are explicitly disabled.
func (s *Store) GetDisabledMetrics() (map[string]bool, error) {
	rows, err := s.db.Query("SELECT metric_name FROM metric_state WHERE enabled = 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result[name] = true
	}
	return result, rows.Err()
}

// SetMetricEnabled upserts the enabled state of a single metric.
func (s *Store) SetMetricEnabled(name string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO metric_state (metric_name, enabled) VALUES (?, ?)
		ON CONFLICT(metric_name) DO UPDATE SET enabled = excluded.enabled`,
		name, enabledInt)
	return err
}

// SetMetricsBulkEnabled sets the enabled state for multiple metrics at once.
func (s *Store) SetMetricsBulkEnabled(names []string, enabled bool) error {
	if len(names) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	stmt, err := tx.Prepare(`
		INSERT INTO metric_state (metric_name, enabled) VALUES (?, ?)
		ON CONFLICT(metric_name) DO UPDATE SET enabled = excluded.enabled`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, name := range names {
		if _, err := stmt.Exec(name, enabledInt); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// --- Dashboard Layouts ---

// GetDashboardLayout returns a layout by ID.
func (s *Store) GetDashboardLayout(id int64) (*model.DashboardLayout, error) {
	row := s.db.QueryRow("SELECT id, name, layout, updated FROM dashboard_layouts WHERE id = ?", id)
	var dl model.DashboardLayout
	if err := row.Scan(&dl.ID, &dl.Name, &dl.Layout, &dl.Updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &dl, nil
}

// ListDashboardLayouts returns all layouts.
func (s *Store) ListDashboardLayouts() ([]model.DashboardLayout, error) {
	rows, err := s.db.Query("SELECT id, name, layout, updated FROM dashboard_layouts ORDER BY updated DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.DashboardLayout
	for rows.Next() {
		var dl model.DashboardLayout
		if err := rows.Scan(&dl.ID, &dl.Name, &dl.Layout, &dl.Updated); err != nil {
			return nil, err
		}
		result = append(result, dl)
	}
	return result, rows.Err()
}

// CreateDashboardLayout inserts a new layout.
func (s *Store) CreateDashboardLayout(dl *model.DashboardLayout) (int64, error) {
	dl.Updated = time.Now().Unix()
	res, err := s.db.Exec("INSERT INTO dashboard_layouts (name, layout, updated) VALUES (?, ?, ?)",
		dl.Name, dl.Layout, dl.Updated)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateDashboardLayout updates an existing layout.
func (s *Store) UpdateDashboardLayout(dl *model.DashboardLayout) error {
	dl.Updated = time.Now().Unix()
	_, err := s.db.Exec("UPDATE dashboard_layouts SET name = ?, layout = ?, updated = ? WHERE id = ?",
		dl.Name, dl.Layout, dl.Updated, dl.ID)
	return err
}

// DeleteDashboardLayout deletes a layout.
func (s *Store) DeleteDashboardLayout(id int64) error {
	_, err := s.db.Exec("DELETE FROM dashboard_layouts WHERE id = ?", id)
	return err
}

// --- Alert Rules ---

// ListAlertRules returns all alert rules.
func (s *Store) ListAlertRules() ([]model.AlertRule, error) {
	rows, err := s.db.Query("SELECT id, metric_pattern, operator, threshold, severity, message_en, message_ko, enabled FROM alert_rules ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.AlertRule
	for rows.Next() {
		var r model.AlertRule
		var enabled int
		if err := rows.Scan(&r.ID, &r.MetricPattern, &r.Operator, &r.Threshold, &r.Severity, &r.MessageEN, &r.MessageKO, &enabled); err != nil {
			return nil, err
		}
		r.Enabled = enabled != 0
		result = append(result, r)
	}
	return result, rows.Err()
}

// CreateAlertRule inserts a new alert rule and returns the ID.
func (s *Store) CreateAlertRule(r *model.AlertRule) (int64, error) {
	enabledInt := 0
	if r.Enabled {
		enabledInt = 1
	}
	res, err := s.db.Exec(
		"INSERT INTO alert_rules (metric_pattern, operator, threshold, severity, message_en, message_ko, enabled) VALUES (?, ?, ?, ?, ?, ?, ?)",
		r.MetricPattern, r.Operator, r.Threshold, r.Severity, r.MessageEN, r.MessageKO, enabledInt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateAlertRule updates an existing alert rule.
func (s *Store) UpdateAlertRule(r *model.AlertRule) error {
	enabledInt := 0
	if r.Enabled {
		enabledInt = 1
	}
	_, err := s.db.Exec(
		"UPDATE alert_rules SET metric_pattern=?, operator=?, threshold=?, severity=?, message_en=?, message_ko=?, enabled=? WHERE id=?",
		r.MetricPattern, r.Operator, r.Threshold, r.Severity, r.MessageEN, r.MessageKO, enabledInt, r.ID)
	return err
}

// DeleteAlertRule deletes an alert rule by ID.
func (s *Store) DeleteAlertRule(id int64) error {
	_, err := s.db.Exec("DELETE FROM alert_rules WHERE id = ?", id)
	return err
}
