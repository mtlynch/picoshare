CREATE TABLE settings (
    id INTEGER PRIMARY KEY,
    default_expiration_in_days INTEGER
);

INSERT INTO settings (
    id,
    default_expiration_in_days
) VALUES (
    1,
    30
);
