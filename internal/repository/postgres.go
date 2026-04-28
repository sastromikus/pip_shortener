package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/model"
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

func (r *PostgresRepository) PutBatchIfAbsent(items []model.URLItem) error {
	if len(items) == 0 {
		return nil
	}

	var b strings.Builder
	args := make([]any, 0, len(items)*2)

	b.WriteString(`INSERT INTO urls (short_id, original_url) VALUES `)

	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}

		argPos := i*2 + 1
		b.WriteString(fmt.Sprintf("($%d, $%d)", argPos, argPos+1))

		args = append(args, item.ID, item.Original)
	}

	b.WriteString(` ON CONFLICT DO NOTHING`)

	_, err := r.db.ExecContext(context.TODO(), b.String(), args...)
	return err
}

func (r *PostgresRepository) AddUserURL(userID, shortID string) error {
	_, err := r.db.ExecContext(context.TODO(),
		`INSERT INTO user_urls (user_id, short_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, short_id) DO NOTHING`,
		userID,
		shortID,
	)

	return err
}

func (r *PostgresRepository) ListUserURLs(userID string) ([]model.URLMapping, error) {
	rows, err := r.db.QueryContext(context.TODO(),
		`SELECT u.short_id, u.original_url
		   FROM user_urls uu
		   JOIN urls u ON u.short_id = uu.short_id
		  WHERE uu.user_id = $1
		  ORDER BY u.id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.URLMapping, 0)
	for rows.Next() {
		var shortID, original string
		if err := rows.Scan(&shortID, &original); err != nil {
			return nil, err
		}

		out = append(out, model.URLMapping{
			ID:       shortID,
			Original: original,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
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
		"migrations/0003_create_user_urls.sql",
	}

	for _, path := range migrations {
		if err := RunSQLMigration(db, path); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}

	return nil
}
