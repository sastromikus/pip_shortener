package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

func RunSQLMigration(db *sql.DB, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if len(b) == 0 {
		return nil
	}

	if _, err := db.Exec(string(b)); err != nil {
		return err
	}

	return nil
}

func RunPostgresMigrations(db *sql.DB) error {
	dirs, err := migrationPaths()
	if err != nil {
		return fmt.Errorf("find migrations: %w", err)
	}

	files := []string{
		"0001_create_urls.sql",
		"0002_unique_original.sql",
		"0003_create_user_urls.sql",
		"0004_add_is_deleted.sql",
	}

	var lastErr error
	for _, dir := range dirs {
		ok := true
		for _, file := range files {
			path := filepath.Join(dir, file)
			if err := RunSQLMigration(db, path); err != nil {
				lastErr = err
				ok = false
				break
			}
		}
		if ok {
			return nil
		}
	}

	return fmt.Errorf("run migrations: %w", lastErr)
}

func migrationPaths() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cwdMigrations := filepath.Join(cwd, "migrations")

	exe, err := os.Executable()
	if err != nil {
		return []string{cwdMigrations}, nil
	}

	exeDir := filepath.Dir(exe)
	return []string{
		cwdMigrations,
		filepath.Join(exeDir, "migrations"),
		filepath.Clean(filepath.Join(exeDir, "..", "..", "migrations")),
	}, nil
}
