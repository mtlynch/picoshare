ALTER TABLE guest_links
RENAME COLUMN expiration_time TO url_expiration_time;

ALTER TABLE guest_links
ADD file_expiration_time TEXT;

UPDATE guest_links
SET file_expiration_time = 'NEVER'
WHERE file_expiration_time IS NULL;
