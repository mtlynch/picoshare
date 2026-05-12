-- Create an index for fast file size calculation.
-- https://github.com/mtlynch/picoshare/issues/220
CREATE INDEX idx_entries_data_length
ON entries_data (id, LENGTH(chunk));
