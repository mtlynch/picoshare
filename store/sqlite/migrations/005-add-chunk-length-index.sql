-- Create an index for fast file size calculation.
-- https://github.com/mtlynch/picoshare/issues/220
CREATE INDEX li1
ON entries_data(id, LENGTH(chunk));
