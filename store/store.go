package store

import (
	"errors"
	"fmt"

	"github.com/mtlynch/picoshare/picoshare"
)

// ErrGuestUploadRejected indicates that a guest link cannot accept an upload.
var ErrGuestUploadRejected = errors.New("guest link no longer permits this upload")

// EntryNotFoundError occurs when no entry exists with the given ID.
type EntryNotFoundError struct {
	ID picoshare.EntryID
}

// GuestLinkNotFoundError occurs when no guest link exists with the given ID.
type GuestLinkNotFoundError struct {
	ID picoshare.GuestLinkID
}

func (f EntryNotFoundError) Error() string {
	return fmt.Sprintf("Could not find entry with ID %v", f.ID)
}

func (f GuestLinkNotFoundError) Error() string {
	return fmt.Sprintf("Could not find guest link with ID %v", f.ID)
}
