ALTER TABLE urls
ADD CONSTRAINT urls_original_url_uq UNIQUE (original_url);