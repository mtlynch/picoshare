-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd


CREATE TABLE IF NOT EXISTS entries (
  id TEXT PRIMARY KEY,
  filename TEXT,
  content_type TEXT,
  upload_time TEXT,
  expiration_time TEXT
  );

CREATE TABLE IF NOT EXISTS entries_data (
  id TEXT,
  chunk_index INTEGER,
  chunk BLOB,
  FOREIGN KEY(id) REFERENCES entries(id)
  );

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd

DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS entries_data;
