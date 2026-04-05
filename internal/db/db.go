package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "sdr.db")
	sqlDB, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	sqlDB.SetMaxOpenConns(1)

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS service_state (
			service_id TEXT PRIMARY KEY,
			last_started DATETIME,
			last_stopped DATETIME,
			auto_start BOOLEAN DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS pager_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			msg_type TEXT,
			address TEXT,
			func_code INTEGER,
			message TEXT,
			frequency TEXT,
			raw_line TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pager_timestamp ON pager_messages(timestamp)`,
		`CREATE TABLE IF NOT EXISTS ais_ships (
			mmsi TEXT PRIMARY KEY,
			name TEXT,
			ship_type INTEGER,
			destination TEXT,
			last_lat REAL,
			last_lon REAL,
			speed REAL,
			course REAL,
			heading INTEGER,
			last_seen DATETIME,
			first_seen DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS ais_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mmsi TEXT NOT NULL,
			lat REAL NOT NULL,
			lon REAL NOT NULL,
			speed REAL,
			course REAL,
			timestamp DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ais_history_mmsi ON ais_history(mmsi)`,
		`CREATE INDEX IF NOT EXISTS idx_ais_history_ts ON ais_history(timestamp)`,
		`CREATE TABLE IF NOT EXISTS gsm_scans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			band TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			result TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS iss_captures (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			filepath TEXT NOT NULL,
			duration INTEGER,
			size INTEGER,
			pass_rise DATETIME,
			pass_set DATETIME,
			max_alt REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			action TEXT NOT NULL,
			detail TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %s: %w", m[:50], err)
		}
	}

	log.Println("[db] Migrations completed successfully")
	return nil
}

// Settings

func (db *DB) GetSetting(key string) (string, error) {
	var val string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value,
	)
	return err
}

// Users

func (db *DB) GetUser(username string) (*User, error) {
	u := &User{}
	err := db.QueryRow(
		"SELECT id, username, password_hash, role FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (db *DB) CreateUser(username, passwordHash, role string) error {
	_, err := db.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		username, passwordHash, role,
	)
	return err
}

func (db *DB) UserCount() (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
