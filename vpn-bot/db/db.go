package db

import (
	"database/sql"
	"embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open opens SQLite and applies embedded migrations once (simple bootstrap).
func Open(path string) (*sql.DB, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := d.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = d.Close()
		return nil, err
	}
	if err := d.Ping(); err != nil {
		_ = d.Close()
		return nil, err
	}
	b, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("read migration: %w", err)
	}
	if _, err := d.Exec(string(b)); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}
