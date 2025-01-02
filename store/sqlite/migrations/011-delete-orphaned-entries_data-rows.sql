-- Delete entries_data rows with no associated entries row. Before #629, it was
-- possible to create orphaned rows, but after #629, it should not be possible.
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
