ALTER TABLE entries_data ADD COLUMN chunk_length INTEGER NOT NULL DEFAULT 0;

UPDATE entries_data SET chunk_length = LENGTH(chunk);

DROP INDEX IF EXISTS idx_entries_data_length;

CREATE INDEX idx_entries_data_chunk_length ON entries_data (id, chunk_length);
