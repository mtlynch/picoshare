CREATE TABLE IF NOT EXISTS guest_links (
    id TEXT PRIMARY KEY,
    label TEXT,
    max_file_size INTEGER,
    uploads_left INTEGER,
    expiration_time TEXT
);

ALTER TABLE entries RENAME TO old_entries;

CREATE TABLE IF NOT EXISTS entries (
    id TEXT PRIMARY KEY,
    filename TEXT,
    content_type TEXT,
    upload_time TEXT,
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
    '' AS guest_link_id
FROM
    old_entries;

DROP TABLE old_entries;
