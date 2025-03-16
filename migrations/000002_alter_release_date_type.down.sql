ALTER TABLE songs
ALTER COLUMN release_date TYPE DATE USING (release_date::DATE);