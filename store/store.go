package store

import (
	"fmt"

	"github.com/mtlynch/picoshare/picoshare"
)

// EntryNotFoundError occurs when no entry exists with the given ID.
type EntryNotFoundError struct {
	ID picoshare.EntryID
}

func (f EntryNotFoundError) Error() string {
	return fmt.Sprintf("Could not find entry with ID %v", f.ID)
}

// GuestLinkNotFoundError occurs when no guest link exists with the given ID.
type GuestLinkNotFoundError struct {
	ID picoshare.GuestLinkID
}

func (f GuestLinkNotFoundError) Error() string {
	return fmt.Sprintf("Could not find guest link with ID %v", f.ID)
}

// GuestUploadRejectedError identifies a guest-link authorization rejection.
type GuestUploadRejectedError struct {
	Reason string
}

func (e GuestUploadRejectedError) Error() string {
	return e.Reason
}

// GuestUploadInvalidError identifies invalid upload data or lifecycle usage.
type GuestUploadInvalidError struct {
	Err error
}

func (e GuestUploadInvalidError) Error() string {
	return e.Err.Error()
}

func (e GuestUploadInvalidError) Unwrap() error {
	return e.Err
}
