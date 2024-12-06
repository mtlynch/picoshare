ALTER TABLE guest_links
RENAME COLUMN expiration_time to url_expiration_time;

ALTER TABLE guest_links
ADD file_expiration_time TEXT;