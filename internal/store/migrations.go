package store

import (
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS events (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	plugin     TEXT NOT NULL,
	event      TEXT NOT NULL,
	timestamp  DATETIME NOT NULL,
	meta       TEXT NOT NULL,
	synced     BOOLEAN DEFAULT FALSE,
	synced_at  DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_events_unsynced ON events(synced) WHERE synced = FALSE;
CREATE INDEX IF NOT EXISTS idx_events_plugin   ON events(plugin, timestamp);

CREATE TABLE IF NOT EXISTS collection_cursors (
	plugin            TEXT PRIMARY KEY,
	last_collected_at DATETIME NOT NULL
);
`

func applyMigrations(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}
