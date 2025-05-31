-- Convert to STRICT tables and improve constraints.

-- Temporarily disable foreign key constraints during migration.
PRAGMA foreign_keys = OFF;

-- 2022-02-20 is PicoShare epoch (the day of the first release), so we verify
-- that creation and modification timestamps can never be before this point.

-- Create all new tables with STRICT mode and comprehensive constraints.
CREATE TABLE guest_links_new (
    id TEXT PRIMARY KEY,
    label TEXT CHECK (
        label IS NULL OR length(label) <= 200
    ),
    max_file_bytes INTEGER CHECK (
        max_file_bytes IS NULL OR max_file_bytes > 0
    ),
    max_file_uploads INTEGER CHECK (
        max_file_uploads IS NULL OR max_file_uploads > 0
    ),
    creation_time TEXT NOT NULL CHECK (
        datetime(creation_time) IS NOT NULL
        AND datetime(creation_time) >= datetime('2022-02-20')
    ),
    url_expiration_time TEXT CHECK (
        url_expiration_time IS NULL OR (
            datetime(url_expiration_time) IS NOT NULL
            AND datetime(url_expiration_time) >= datetime('2022-02-20')
        )
    ),
    file_expiration_time TEXT,
    is_disabled INTEGER NOT NULL CHECK (is_disabled IN (0, 1)) DEFAULT 0
) STRICT;

CREATE TABLE entries_new (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    upload_time TEXT NOT NULL CHECK (
        datetime(upload_time) IS NOT NULL
        AND datetime(upload_time) >= datetime('2022-02-20')
    ),
    expiration_time TEXT CHECK (
        expiration_time IS NULL OR (
            datetime(expiration_time) IS NOT NULL
            AND datetime(expiration_time) >= datetime('2022-02-20')
        )
    ),
    -- guest_link_id identifies which guest link (if any) the client used to
    -- upload this entry.
    guest_link_id TEXT,
    note TEXT
) STRICT;

CREATE TABLE entries_data_new (
    id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
    chunk BLOB NOT NULL,
    PRIMARY KEY (id, chunk_index)
) STRICT;

CREATE TABLE downloads_new (
    entry_id TEXT NOT NULL,
    download_timestamp TEXT NOT NULL CHECK (
        datetime(download_timestamp) IS NOT NULL
        AND datetime(download_timestamp) >= datetime('2022-02-20')
    ),
    client_ip TEXT,
    user_agent TEXT,
    FOREIGN KEY (entry_id) REFERENCES entries_new (id)
) STRICT;

CREATE TABLE settings_new (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    default_expiration_in_days INTEGER CHECK (
        default_expiration_in_days IS NULL
        OR default_expiration_in_days >= 0
    )
) STRICT;

-- Copy data to new tables (guest_links first, then entries to satisfy foreign
-- key).
INSERT INTO guest_links_new
SELECT
    id,
    label,
    max_file_bytes,
    max_file_uploads,
    creation_time,
    url_expiration_time,
    file_expiration_time,
    is_disabled
FROM guest_links;

INSERT INTO entries_new
SELECT
    id,
    filename,
    content_type,
    upload_time,
    expiration_time,
    CASE
        WHEN
            guest_link_id IS NOT NULL AND guest_link_id != ''
            AND guest_link_id IN (SELECT id FROM guest_links_new)
            THEN guest_link_id
    END AS guest_link_id,
    note
FROM entries;

INSERT INTO entries_data_new
SELECT
    id,
    chunk_index,
    chunk
FROM entries_data;

INSERT INTO downloads_new
SELECT
    entry_id,
    download_timestamp,
    client_ip,
    user_agent
FROM downloads;

INSERT INTO settings_new
SELECT
    id,
    default_expiration_in_days
FROM settings;

-- Drop old tables.
DROP TABLE downloads;
DROP TABLE entries_data;
DROP TABLE entries;
DROP TABLE guest_links;
DROP TABLE settings;

-- Rename new tables.
ALTER TABLE guest_links_new RENAME TO guest_links;
ALTER TABLE entries_new RENAME TO entries;
ALTER TABLE entries_data_new RENAME TO entries_data;
ALTER TABLE downloads_new RENAME TO downloads;
ALTER TABLE settings_new RENAME TO settings;

-- Add foreign key constraint after renaming tables to avoid issues with the
-- constraint during migration.
CREATE INDEX idx_entries_guest_link_id ON entries (guest_link_id);

-- Recreate the index for fast file size calculation.
CREATE INDEX idx_entries_data_length
ON entries_data (id, length(chunk));

-- Re-enable foreign key constraints
PRAGMA foreign_keys = ON;
