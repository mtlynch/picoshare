CREATE TABLE IF NOT EXISTS guest_links (
    id TEXT PRIMARY KEY,
    label TEXT,
    max_file_bytes INTEGER,
    uploads_left INTEGER,
    creation_time TEXT NOT NULL,
    expiration_time TEXT
);

ALTER TABLE entries RENAME TO old_entries;

CREATE TABLE IF NOT EXISTS entries (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    upload_time TEXT NOT NULL,
    expiration_time TEXT,
    guest_link_id TEXT,
    FOREIGN KEY(guest_link_id) REFERENCES guest_links(id)
);

INSERT INTO entries
SELECT
    id,
    filename,
    content_type,
    upload_time,
    expiration_time,
    NULL AS guest_link_id
FROM
    old_entries;

DROP TABLE old_entries;
