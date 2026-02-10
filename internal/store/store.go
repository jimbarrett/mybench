package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps the local SQLite database.
type Store struct {
	db *sql.DB
}

// New opens or creates the SQLite database in the user's config directory.
func New() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "mybench.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS app_config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS connections (
			id            TEXT PRIMARY KEY,
			name          TEXT NOT NULL,
			host          TEXT NOT NULL,
			port          INTEGER NOT NULL DEFAULT 3306,
			username      TEXT NOT NULL,
			password      TEXT NOT NULL DEFAULT '',
			default_db    TEXT NOT NULL DEFAULT '',
			use_ssl       INTEGER NOT NULL DEFAULT 0,
			ssh_enabled   INTEGER NOT NULL DEFAULT 0,
			ssh_host      TEXT NOT NULL DEFAULT '',
			ssh_port      INTEGER NOT NULL DEFAULT 22,
			ssh_user      TEXT NOT NULL DEFAULT '',
			ssh_auth      TEXT NOT NULL DEFAULT 'key',
			ssh_key_path  TEXT NOT NULL DEFAULT '',
			ssh_password  TEXT NOT NULL DEFAULT '',
			sort_order    INTEGER NOT NULL DEFAULT 0,
			created_at    TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	return err
}

// GetConfig retrieves a config value by key.
func (s *Store) GetConfig(key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM app_config WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetConfig sets a config key-value pair.
func (s *Store) SetConfig(key, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO app_config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "mybench"), nil
}
