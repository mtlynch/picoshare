-- Delete entries_data records that reference non-existent entries.
-- This can happen if entries were deleted but their data chunks weren't
-- cleaned up.
DELETE FROM entries_data
WHERE id NOT IN (
    SELECT id
    FROM entries
);

-- Delete download records that reference non-existent entries.
-- This can happen if entries were deleted but their download history wasn't
-- cleaned up.
DELETE FROM downloads
WHERE entry_id NOT IN (
    SELECT id
    FROM entries
);

-- Set guest_link_id to NULL for entries that reference non-existent guest
-- links.
-- This can happen if guest links were deleted but entries still reference
-- them.
UPDATE entries
SET guest_link_id = NULL
WHERE
    guest_link_id IS NOT NULL
    AND guest_link_id NOT IN (
        SELECT id
        FROM guest_links
    );
