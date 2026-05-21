package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Get(id string) (string, bool) {
	var original string

	err := r.db.QueryRowContext(context.TODO(),
		`SELECT original_url FROM urls WHERE short_id = $1`,
		id,
	).Scan(&original)

	if err != nil {
		return "", false
	}

	return original, true
}

func (r *PostgresRepository) GetByOriginal(original string) (string, bool) {
	var shortID string

	err := r.db.QueryRowContext(context.TODO(),
		`SELECT short_id FROM urls WHERE original_url = $1`,
		original,
	).Scan(&shortID)

	if err != nil {
		return "", false
	}

	return shortID, true
}

func (r *PostgresRepository) PutIfAbsent(id string, original string) (bool, error) {
	res, err := r.db.ExecContext(context.TODO(),
		`INSERT INTO urls (short_id, original_url)
		 VALUES ($1, $2)
		 ON CONFLICT (short_id) DO NOTHING`,
		id,
		original,
	)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

func NewPostgresStorage(ctx context.Context, dsn string) (*sql.DB, *PostgresRepository, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := RunPostgresMigrations(db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("run postgres migrations: %w", err)
	}

	return db, NewPostgresRepository(db), nil
}

func RunPostgresMigrations(db *sql.DB) error {
	migrations := []string{
		"migrations/0001_create_urls.sql",
		"migrations/0002_unique_original.sql",
	}

	for _, path := range migrations {
		if err := RunSQLMigration(db, path); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}

	return nil
}
