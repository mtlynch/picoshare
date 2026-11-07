package store

import (
	"fmt"

	"github.com/mtlynch/picoshare/picoshare"
)

// EntryNotFoundError occurs when no entry exists with the given ID.
type EntryNotFoundError struct {
	ID picoshare.EntryID
}

// GuestLinkNotFoundError occurs when no guest link exists with the given ID.
type GuestLinkNotFoundError struct {
	ID picoshare.GuestLinkID
}

// GuestUploadRejectedError occurs when a guest link cannot accept an upload.
type GuestUploadRejectedError struct {
	ID picoshare.GuestLinkID
}

func (f EntryNotFoundError) Error() string {
	return fmt.Sprintf("Could not find entry with ID %v", f.ID)
}

func (f GuestLinkNotFoundError) Error() string {
	return fmt.Sprintf("Could not find guest link with ID %v", f.ID)
}

func (e GuestUploadRejectedError) Error() string {
	return fmt.Sprintf("Guest link %s no longer permits this upload", e.ID)
}
