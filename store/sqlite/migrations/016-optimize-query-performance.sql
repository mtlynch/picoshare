-- Optimize query performance with strategic indexes and cached columns

-- Add strategic indexes for common query patterns
CREATE INDEX idx_entries_upload_time_id ON entries(upload_time DESC, id);
CREATE INDEX idx_entries_expiration_time ON entries(expiration_time);
CREATE INDEX idx_downloads_entry_timestamp ON downloads(entry_id, download_timestamp DESC);

-- Add cached file_size column to entries table for performance
ALTER TABLE entries ADD COLUMN file_size INTEGER;

-- Populate existing entries with calculated file sizes
UPDATE entries SET file_size = (
    SELECT SUM(LENGTH(chunk))
    FROM entries_data
    WHERE entries_data.id = entries.id
);

-- Add cached download_count column for faster metadata queries
ALTER TABLE entries ADD COLUMN download_count INTEGER DEFAULT 0;

-- Populate existing download counts
UPDATE entries SET download_count = (
    SELECT COUNT(*)
    FROM downloads
    WHERE downloads.entry_id = entries.id
);
