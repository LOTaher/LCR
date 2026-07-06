package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func Init(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	);`)
	if err != nil {
		return err
	}

	names, err := getMigrations()
	if err != nil {
		return err
	}

	for _, name := range names {
		applied, err := isMigrationApplied(db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := applyMigration(db, name); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}

		fmt.Printf("Applied migration %s\n", name)
	}

	fmt.Println("Database initialized")

	return nil
}

func getMigrations() ([]string, error) {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	sort.Strings(names)

	return names, nil
}

func isMigrationApplied(db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func applyMigration(db *sql.DB, name string) error {
	contents, err := os.ReadFile(filepath.Join("migrations", name))
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(contents)); err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)", name, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	return tx.Commit()
}
