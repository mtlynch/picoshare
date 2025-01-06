ALTER TABLE guest_links ADD COLUMN is_disabled INTEGER DEFAULT 0;

UPDATE guest_links SET is_disabled = 0;