package store

import "database/sql"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS metric_samples (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		collector TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		value REAL NOT NULL,
		labels TEXT DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_samples_name_ts ON metric_samples(metric_name, timestamp);
	CREATE INDEX IF NOT EXISTS idx_samples_ts ON metric_samples(timestamp);`,

	`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS collector_state (
		collector_id TEXT PRIMARY KEY,
		enabled INTEGER NOT NULL DEFAULT 0,
		config_json TEXT DEFAULT '{}'
	);`,

	`CREATE TABLE IF NOT EXISTS dashboard_layouts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL DEFAULT 'default',
		layout TEXT NOT NULL,
		updated INTEGER NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS metric_state (
		metric_name TEXT PRIMARY KEY,
		enabled INTEGER NOT NULL DEFAULT 1
	);`,

	`CREATE TABLE IF NOT EXISTS alert_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		metric_pattern TEXT NOT NULL,
		operator TEXT NOT NULL DEFAULT 'gt',
		threshold REAL NOT NULL,
		severity TEXT NOT NULL DEFAULT 'warning',
		message_en TEXT NOT NULL DEFAULT '',
		message_ko TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1
	);`,
}

func runMigrations(db *sql.DB) error {
	// Create migration tracking table
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return err
	}

	var currentVersion int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&currentVersion); err != nil {
		return err
	}

	for i := currentVersion; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", i+1); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
