ALTER TABLE entries ADD COLUMN file_size INTEGER;

-- Populate file sizes for legacy entries
UPDATE
  entries
SET
  file_size = (
    SELECT
        SUM(LENGTH(chunk)) AS file_size
      FROM
        entries_data
      WHERE
        entries.id = entries_data.id
  );
