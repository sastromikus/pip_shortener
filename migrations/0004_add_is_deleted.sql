-- +goose Up
ALTER TABLE urls
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS urls_is_deleted_idx ON urls(is_deleted);

-- +goose Down
DROP INDEX IF EXISTS urls_is_deleted_idx;

ALTER TABLE urls
DROP COLUMN IF EXISTS is_deleted;
