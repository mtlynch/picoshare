-- Create new table with correct foreign key
CREATE TABLE new_entries_data (
    id TEXT,
    chunk_index INTEGER,
    chunk BLOB,
    FOREIGN KEY (id) REFERENCES entries (id)
);

-- Copy all data from old table to new table
INSERT INTO new_entries_data
SELECT
    id,
    chunk_index,
    chunk
FROM entries_data;

-- Drop old table
DROP TABLE entries_data;

-- Rename new table to original name
ALTER TABLE new_entries_data RENAME TO entries_data;

-- Recreate the index for file size calculation
--CREATE INDEX idx_entries_data_length
--ON entries_data (id, LENGTH(chunk));

-- TODO: Explain this.
DELETE FROM
    entries_data
WHERE
    id
    IN (
        SELECT DISTINCT entries_data.id AS entry_id
        FROM
            entries_data
        LEFT JOIN
            entries ON entries_data.id = entries.id
        WHERE
            entries.id IS NULL
    );
