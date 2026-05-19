CREATE TABLE friendly_links (
    friendly_name TEXT PRIMARY KEY,
    id TEXT NOT NULL,
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
    is_disabled INTEGER NOT NULL CHECK (is_disabled IN (0, 1)) DEFAULT 0,
    FOREIGN KEY (id) REFERENCES entries (id) ON DELETE CASCADE
) STRICT;
