CREATE TABLE IF NOT EXISTS entries_data (
    id TEXT,
    chunk_index INTEGER,
    chunk BLOB --,
    FOREIGN KEY(id) REFERENCES entries(id)
);
