-- Store the file size in the entries table. The authoritative size is in the
-- entries_data table, but it's too slow to recalculate that at runtime because
-- the SQLite LENGTH() function reads the full data into memory. For databases
-- bigger than a few hundred MB, this causes a noticeable slowdown.
ALTER TABLE entries ADD COLUMN file_size INTEGER;

-- Populate file sizes for legacy entries
UPDATE entries
SET file_size = (
    SELECT SUM(LENGTH(entries_data.chunk)) AS file_size
    FROM
        entries_data
    WHERE
        entries.id = entries_data.id
);
