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

func (r *PostgresRepository) GetWithDeleted(id string) (string, bool, bool) {
	var original string
	var deleted bool

	err := r.db.QueryRowContext(context.TODO(),
		`SELECT original_url, is_deleted FROM urls WHERE short_id = $1`,
		id,
	).Scan(&original, &deleted)
	if err != nil {
		return "", false, false
	}

	return original, true, deleted
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
		 ON CONFLICT DO NOTHING`,
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

// Put is kept for tests and simple compatibility. Business code should prefer PutIfAbsent.
func (r *PostgresRepository) Put(id string, original string) {
	_, _ = r.PutIfAbsent(id, original)
}

func (r *PostgresRepository) Exists(id string) bool {
	_, ok := r.Get(id)
	return ok
}

func (r *PostgresRepository) AddUserURL(userID, shortID string) error {
	return r.AddUserURLs(userID, []string{shortID})
}

func (r *PostgresRepository) AddUserURLs(userID string, shortIDs []string) error {
	if userID == "" || len(shortIDs) == 0 {
		return nil
	}

	var b strings.Builder
	args := make([]any, 0, len(shortIDs)*2)

	b.WriteString(`INSERT INTO user_urls (user_id, short_id) VALUES `)
	for i, shortID := range shortIDs {
		if i > 0 {
			b.WriteString(", ")
		}

		argPos := i*2 + 1
		b.WriteString(fmt.Sprintf("($%d, $%d)", argPos, argPos+1))
		args = append(args, userID, shortID)
	}
	b.WriteString(` ON CONFLICT (user_id, short_id) DO NOTHING`)

	_, err := r.db.ExecContext(context.TODO(), b.String(), args...)
	return err
}

func (r *PostgresRepository) ListUserURLs(userID string) ([]model.UserURL, error) {
	rows, err := r.db.QueryContext(context.TODO(),
		`SELECT u.short_id, u.original_url
		   FROM user_urls uu
		   JOIN urls u ON u.short_id = uu.short_id
		  WHERE uu.user_id = $1
		    AND u.is_deleted = FALSE
		  ORDER BY u.id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.UserURL, 0)
	for rows.Next() {
		var item model.UserURL
		if err := rows.Scan(&item.ShortID, &item.Original); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *PostgresRepository) MarkDeleted(userID string, ids []string) error {
	if userID == "" || len(ids) == 0 {
		return nil
	}

	var b strings.Builder
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)

	b.WriteString(`UPDATE urls u
		   SET is_deleted = TRUE
		  FROM user_urls uu
		 WHERE uu.short_id = u.short_id
		   AND uu.user_id = $1
		   AND u.short_id IN (`)

	for i, id := range ids {
		if i > 0 {
			b.WriteString(", ")
		}
		argPos := i + 2
		b.WriteString(fmt.Sprintf("$%d", argPos))
		args = append(args, id)
	}
	b.WriteString(`)`)

	_, err := r.db.ExecContext(context.TODO(), b.String(), args...)
	return err
}
