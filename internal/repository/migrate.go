package repository

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/pressly/goose/v3"
)

const gooseTableName = "goose_db_version"

// RunPostgresMigrations applies PostgreSQL migrations.
func RunPostgresMigrations(ctx context.Context, db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := resetGooseVersionIfSchemaWasWiped(ctx, db); err != nil {
		return fmt.Errorf("check goose schema state: %w", err)
	}

	migrationDir, err := migrationsDir()
	if err != nil {
		return err
	}

	if err := goose.UpContext(ctx, db, migrationDir); err != nil {
		return fmt.Errorf("run postgres migrations: %w", err)
	}

	return nil
}

func migrationsDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("detect migrations dir")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations")), nil
}

func resetGooseVersionIfSchemaWasWiped(ctx context.Context, db *sql.DB) error {
	var exists bool

	err := db.QueryRowContext(ctx, `
		SELECT to_regclass('public.urls') IS NOT NULL
	`).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	_, err = db.ExecContext(ctx, `DROP TABLE IF EXISTS `+gooseTableName)
	return err
}
