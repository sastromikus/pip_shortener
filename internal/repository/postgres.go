package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/model"
)

// PostgresRepository stores URLs in PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a PostgreSQL repository around an existing DB handle.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// NewPostgresStorage opens PostgreSQL storage and applies migrations.
func NewPostgresStorage(ctx context.Context, dsn string) (*sql.DB, *PostgresRepository, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := RunPostgresMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("run postgres migrations: %w", err)
	}

	return db, NewPostgresRepository(db), nil
}

// Get returns the original URL by short id.
func (r *PostgresRepository) Get(ctx context.Context, id string) (string, bool) {
	var original string
	err := r.db.QueryRowContext(ctx,
		`SELECT original_url FROM urls WHERE short_id = $1`,
		id,
	).Scan(&original)
	if err != nil {
		return "", false
	}

	return original, true
}

// GetWithDeleted returns the original URL and deletion state by short id.
func (r *PostgresRepository) GetWithDeleted(ctx context.Context, id string) (string, bool, bool) {
	var original string
	var deleted bool

	err := r.db.QueryRowContext(ctx,
		`SELECT original_url, is_deleted FROM urls WHERE short_id = $1`,
		id,
	).Scan(&original, &deleted)
	if err != nil {
		return "", false, false
	}

	return original, true, deleted
}

// GetByOriginal returns the short id for an original URL.
func (r *PostgresRepository) GetByOriginal(ctx context.Context, original string) (string, bool) {
	var shortID string
	err := r.db.QueryRowContext(ctx,
		`SELECT short_id FROM urls WHERE original_url = $1`,
		original,
	).Scan(&shortID)
	if err != nil {
		return "", false
	}

	return shortID, true
}

// PutIfAbsent stores a URL only when neither short id nor original URL exists.
func (r *PostgresRepository) PutIfAbsent(ctx context.Context, id string, original string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
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

// PutBatchIfAbsent stores multiple URL records when they are absent.
func (r *PostgresRepository) PutBatchIfAbsent(ctx context.Context, items []model.URLItem) error {
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

	_, err := r.db.ExecContext(ctx, b.String(), args...)
	return err
}

// Put is kept for tests and simple compatibility. Business code should prefer PutIfAbsent.
func (r *PostgresRepository) Put(id string, original string) {
	_, _ = r.PutIfAbsent(context.Background(), id, original)
}

// Exists reports whether a short id is already stored.
func (r *PostgresRepository) Exists(id string) bool {
	_, ok := r.Get(context.Background(), id)
	return ok
}

// AddUserURL associates a short id with a user.
func (r *PostgresRepository) AddUserURL(ctx context.Context, userID, shortID string) error {
	return r.AddUserURLs(ctx, userID, []string{shortID})
}

// AddUserURLs associates multiple short ids with a user.
func (r *PostgresRepository) AddUserURLs(ctx context.Context, userID string, shortIDs []string) error {
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

	_, err := r.db.ExecContext(ctx, b.String(), args...)
	return err
}

// ListUserURLs returns all non-deleted URLs owned by a user.
func (r *PostgresRepository) ListUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	rows, err := r.db.QueryContext(ctx,
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

// MarkDeleted marks user-owned URLs as deleted.
func (r *PostgresRepository) MarkDeleted(ctx context.Context, userID string, ids []string) error {
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

	_, err := r.db.ExecContext(ctx, b.String(), args...)
	return err
}
