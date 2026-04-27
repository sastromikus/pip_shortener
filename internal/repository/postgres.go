package repository

import (
	"context"
	"database/sql"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Get(id string) (string, bool) {
	var original string
	err := r.db.QueryRowContext(context.Background(),
		`SELECT original_url FROM urls WHERE short_id = $1`, id,
	).Scan(&original)
	if err != nil {
		return "", false
	}
	return original, true
}

func (r *PostgresRepository) Put(id string, original string) {
	_, _ = r.db.ExecContext(context.Background(),
		`INSERT INTO urls (short_id, original_url) VALUES ($1, $2)
		 ON CONFLICT (short_id) DO UPDATE SET original_url = EXCLUDED.original_url`,
		id, original,
	)
}

func (r *PostgresRepository) Exists(id string) bool {
	var exists bool
	err := r.db.QueryRowContext(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM urls WHERE short_id = $1)`, id,
	).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}