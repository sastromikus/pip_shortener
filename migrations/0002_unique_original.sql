DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'urls_original_url_uq'
  ) THEN
    ALTER TABLE urls
      ADD CONSTRAINT urls_original_url_uq UNIQUE (original_url);
  END IF;
END $$;