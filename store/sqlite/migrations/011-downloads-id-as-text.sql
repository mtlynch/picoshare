-- In migration 007, we accidentally specified entry_id as an INTEGER instead of
-- TEXT. This migration fixes the table so that entry_id has a correct TEXT
-- type.
CREATE TABLE new_downloads (
    entry_id TEXT,
    download_timestamp TEXT,
    client_ip TEXT,
    user_agent TEXT,
    FOREIGN KEY (entry_id) REFERENCES entries (id)
);

INSERT INTO new_downloads
SELECT
    CAST(entry_id AS TEXT) AS entry_id,
    download_timestamp,
    client_ip,
    user_agent
FROM downloads
WHERE entry_id IN (SELECT id FROM entries);

DROP TABLE downloads;

ALTER TABLE new_downloads RENAME TO downloads;
