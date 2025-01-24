-- In migration 007, we accidentally specified entry_id as an INTEGER instead of
-- TEXT.
CREATE TABLE new_downloads (
    entry_id TEXT,
    download_timestamp TEXT,
    client_ip TEXT,
    user_agent TEXT,
    FOREIGN KEY (entry_id) REFERENCES entries (id)
);

INSERT INTO new_downloads
SELECT
    CAST(entry_id AS TEXT),
    download_timestamp,
    client_ip,
    user_agent
FROM downloads;

DROP TABLE downloads;

ALTER TABLE new_downloads RENAME TO downloads;
