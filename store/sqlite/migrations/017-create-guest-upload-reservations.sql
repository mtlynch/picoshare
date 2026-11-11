-- Reservations intentionally outlive deletion of their guest link so cleanup
-- retains ownership of any staged chunks.
CREATE TABLE guest_upload_reservations (
    entry_id TEXT PRIMARY KEY,
    guest_link_id TEXT NOT NULL,
    max_file_bytes INTEGER CHECK (
        max_file_bytes IS NULL OR max_file_bytes > 0
    ),
    state TEXT NOT NULL DEFAULT 'reserved' CHECK (
        state IN ('reserved', 'writing', 'written')
    )
) STRICT;

CREATE INDEX idx_guest_upload_reservations_guest_link_id
ON guest_upload_reservations (guest_link_id);
