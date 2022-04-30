DROP TABLE entries_data; -- TODO: Do the proper migration


CREATE TABLE IF NOT EXISTS entries_data (
    id TEXT,
    chunk_index INTEGER,
    chunk BLOB,
    -- Store the length explicitly because otherwise SQLite has
    -- to read the full chunk data into memory to calculate it.
    chunk_size INTEGER GENERATED ALWAYS AS (LENGTH(chunk)) STORED,
    FOREIGN KEY(id) REFERENCES entries(id)
);
