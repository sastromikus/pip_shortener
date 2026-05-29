package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

const gooseTableName = "goose_db_version"

func RunPostgresMigrations(ctx context.Context, db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := resetGooseVersionIfSchemaWasWiped(ctx, db); err != nil {
		return fmt.Errorf("check goose schema state: %w", err)
	}

	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("run postgres migrations: %w", err)
	}

	return nil
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
