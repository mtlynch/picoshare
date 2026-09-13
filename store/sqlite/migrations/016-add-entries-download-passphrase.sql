-- download_passphrase is NULL when downloads do not require a passphrase.
-- A non-NULL passphrase must contain between 1 and 100 Unicode code points,
-- matching picoshare.MaxPassphraseCodePoints.
ALTER TABLE entries ADD COLUMN download_passphrase TEXT CHECK (
    download_passphrase IS NULL
    OR length(download_passphrase) BETWEEN 1 AND 100
);
