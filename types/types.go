package types

import (
	"io"
	"time"
)

type EntryID string
type Filename string
type ContentType string
type ExpirationTime time.Time

// Treat a distant expiration time as sort of a sentinel value signifying a "never expire" option.
var NeverExpire = ExpirationTime(time.Date(3000, time.January, 0, 0, 0, 0, 0, time.UTC))

type (
	UploadMetadata struct {
		ID          EntryID
		Filename    Filename
		ContentType ContentType
		Uploaded    time.Time
		Expires     ExpirationTime
		Size        int
	}

	UploadEntry struct {
		UploadMetadata
		Reader io.ReadSeeker
	}

	GuestLinkID           string
	GuestLinkLabel        string
	GuestUploadSizeLimit  uint64
	GuestUploadCountLimit int

	GuestLink struct {
		ID                   GuestLinkID
		Label                GuestLinkLabel
		Created              time.Time
		LastUsed             time.Time
		Expires              ExpirationTime
		UploadSizeRemaining  *GuestUploadSizeLimit
		UploadCountRemaining *GuestUploadCountLimit
	}
)
