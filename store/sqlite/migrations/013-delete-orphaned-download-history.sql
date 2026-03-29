-- Delete download records that reference non-existent entries.
-- Prior to #661, we weren't deleting the download history when we deleted the
-- associated file.
DELETE FROM downloads
WHERE entry_id NOT IN (
    SELECT id
    FROM entries
);
