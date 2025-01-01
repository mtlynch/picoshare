-- We were accidentally writing empty strings, which violated the FOREIGN KEY
-- constraint.
UPDATE entries
SET guest_link_id = NULL
WHERE guest_link_id = '';
