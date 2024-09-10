CREATE TABLE downloads (
    entry_id INTEGER,
    download_timestamp TEXT,
    client_ip TEXT,
    user_agent TEXT,
    FOREIGN KEY (entry_id) REFERENCES entries (id)
);
