-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS urls_original_url_idx
ON urls (original_url);

-- +goose Down
DROP INDEX IF EXISTS urls_original_url_idx;
