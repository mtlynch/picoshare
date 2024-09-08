CREATE TABLE IF NOT EXISTS entries_temp (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    contents BLOB,
    content_type TEXT NOT NULL,
    note TEXT,
    upload_time TEXT NOT NULL,
    expiration_time TEXT,
    guest_link_id TEXT,
    FOREIGN KEY(guest_link_id) REFERENCES guest_links(id)
);

-- Copy data from the existing entries table.
INSERT INTO entries_temp (
    id,
    filename,
    content_type,
    note,
    upload_time,
    expiration_time)
SELECT
    id,
    filename,
    content_type,
    note,
    upload_time,
    expiration_time
FROM entries;

-- Combine and insert data from entries_data.
UPDATE entries_temp
SET contents = (
    SELECT
        GROUP_CONCAT(entries_data.chunk, '' ORDER BY entries_data.chunk_index)
    FROM entries_data
    WHERE entries_data.id = entries_temp.id
);

DROP TABLE entries;
DROP TABLE entries_data;
ALTER TABLE entries_temp RENAME TO entries;
