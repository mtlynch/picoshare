CREATE TABLE IF NOT EXISTS entries (
    id TEXT PRIMARY KEY,
    filename TEXT,
    content_type TEXT,
    upload_time TEXT,
    expiration_time TEXT
);
