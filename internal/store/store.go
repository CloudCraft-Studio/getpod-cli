package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CloudCraft-Studio/getpod-cli/internal/plugin"
	_ "modernc.org/sqlite"
)

// Store persiste eventos de plugins en SQLite.
type Store struct {
	db *sql.DB
}

// StoredEvent es plugin.Metric enriquecido con los campos de persistencia.
type StoredEvent struct {
	ID        int64
	plugin.Metric
	Synced    bool
	SyncedAt  *time.Time
	CreatedAt time.Time
}

// DefaultDBPath retorna la ruta por defecto: ~/.getpod/getpod.db
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".getpod", "getpod.db")
	}
	return filepath.Join(home, ".getpod", "getpod.db")
}

// NewStore abre (o crea) la DB en dbPath, habilita WAL mode y aplica el schema.
func NewStore(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("creating store dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting pragmas: %w", err)
	}

	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close cierra la conexión a la DB.
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveEvents persiste un batch de métricas en una transacción. Meta se serializa como JSON.
func (s *Store) SaveEvents(ctx context.Context, events []plugin.Metric) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO events (plugin, event, timestamp, meta) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing insert: %w", err)
	}
	defer stmt.Close()

	for _, m := range events {
		meta, err := encodeMeta(m.Meta)
		if err != nil {
			return fmt.Errorf("encoding meta for event %q: %w", m.Event, err)
		}
		if _, err := stmt.ExecContext(ctx, m.Plugin, m.Event, encodeTime(m.Timestamp), meta); err != nil {
			return fmt.Errorf("inserting event: %w", err)
		}
	}

	return tx.Commit()
}

// GetUnsynced retorna hasta limit eventos con synced = FALSE, ordenados por id ASC.
func (s *Store) GetUnsynced(ctx context.Context, limit int) ([]StoredEvent, error) {
	panic("not implemented")
}

// MarkSynced marca un batch de eventos como sincronizados.
func (s *Store) MarkSynced(ctx context.Context, ids []int64, syncedAt time.Time) error {
	panic("not implemented")
}

// GetCursor retorna el último timestamp de colección del plugin.
// Si no existe cursor, retorna time.Time{} sin error.
func (s *Store) GetCursor(ctx context.Context, pluginName string) (time.Time, error) {
	panic("not implemented")
}

// SetCursor hace upsert del cursor de colección del plugin.
func (s *Store) SetCursor(ctx context.Context, pluginName string, t time.Time) error {
	panic("not implemented")
}

// encodeTime formatea un time.Time como RFC3339 UTC.
func encodeTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// decodeTime parsea un RFC3339 string. Fallback para el formato "2006-01-02 15:04:05" de SQLite DEFAULT CURRENT_TIMESTAMP.
func decodeTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
}

// encodeMeta serializa map[string]string a JSON.
func encodeMeta(meta map[string]string) (string, error) {
	b, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("encoding meta: %w", err)
	}
	return string(b), nil
}

// decodeMeta deserializa un JSON string a map[string]string.
func decodeMeta(s string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("decoding meta: %w", err)
	}
	return m, nil
}
