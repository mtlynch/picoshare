-- Optimize query performance with strategic indexes and separate cache table

-- Add strategic indexes for common query patterns
CREATE INDEX idx_entries_upload_time_id ON entries (upload_time DESC, id);
CREATE INDEX idx_entries_expiration_time ON entries (expiration_time);
CREATE INDEX idx_downloads_entry_timestamp ON downloads (
    entry_id, download_timestamp DESC
);

-- Create separate cache table for computed values
CREATE TABLE entry_cache (
    entry_id TEXT PRIMARY KEY,
    file_size INTEGER NOT NULL,
    download_count INTEGER NOT NULL DEFAULT 0,
    last_updated TEXT NOT NULL CHECK (
        datetime(last_updated) IS NOT NULL
        AND datetime(last_updated) > datetime('2022-02-19')
    ),
    FOREIGN KEY (entry_id) REFERENCES entries (id) ON DELETE CASCADE
) STRICT;

-- Create index for efficient cache lookups
CREATE INDEX idx_entry_cache_last_updated ON entry_cache (last_updated);

-- Populate cache with existing data
INSERT INTO entry_cache (entry_id, file_size, download_count, last_updated)
WITH fs AS (
    SELECT
        id,
        sum(length(chunk)) AS file_size
    FROM entries_data
    GROUP BY id
),

dc AS (
    SELECT
        entry_id,
        count(*) AS download_count
    FROM downloads
    GROUP BY entry_id
)

SELECT
    e.id,
    coalesce(fs.file_size, 0) AS file_size,
    coalesce(dc.download_count, 0) AS download_count,
    datetime('now') AS last_updated
FROM entries AS e
LEFT JOIN fs ON e.id = fs.id
LEFT JOIN dc ON e.id = dc.entry_id;
