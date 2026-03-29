-- In migration 003, we didn't update entries_data to point to the new entries
-- table, so we have to fix it here. For some reason, the sqlite3 database
-- driver we were using at the time (mattn/go-sqlite3) didn't notice the
-- violation of the FOREIGN KEY constraint.
CREATE TABLE new_entries_data (
    id TEXT,
    chunk_index INTEGER,
    chunk BLOB,
    FOREIGN KEY (id) REFERENCES entries (id)
);

INSERT INTO new_entries_data
SELECT
    id,
    chunk_index,
    chunk
FROM entries_data
WHERE id IN (SELECT id FROM entries);

DROP TABLE entries_data;

ALTER TABLE new_entries_data RENAME TO entries_data;
