package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yaitoo/sqle"
)

// Global database handle
var db *sqle.DB

func initDB() (*sqle.DB, error) {
	dbPath := "./xun-web.db"

	// Create directory if not exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	sqlDB, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	sqlDB.SetMaxOpenConns(1) // SQLite recommended setting
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return sqle.Open(sqlDB), nil
}
