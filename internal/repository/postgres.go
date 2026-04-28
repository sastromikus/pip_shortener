package repository

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
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

func (r *PostgresRepository) GetByOriginal(original string) (string, bool) {
    var shortID string
    err := r.db.QueryRowContext(context.Background(),
        `SELECT short_id FROM urls WHERE original_url = $1`, original,
    ).Scan(&shortID)

    if err != nil {
        return "", false
    }

    return shortID, true
}

func (r *PostgresRepository) Insert(id, original string) error {
	_, err := r.db.ExecContext(context.Background(),
		`INSERT INTO urls (short_id, original_url) VALUES ($1,$2)`,
		id, original,
	)

	return err
}

func IsUniqueViolationOn(err error, constraint string) bool {
	pqe, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	
	if string(pqe.Code) != "23505" {
		return false
	}

	return pqe.Constraint == constraint
}