ALTER TABLE guest_links
RENAME COLUMN expiration_time TO url_expiration_time;

ALTER TABLE guest_links
ADD file_expiration_time TEXT;
