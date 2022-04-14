package types

import (
	"io"
	"time"
)

type (
	EntryID        string
	Filename       string
	ContentType    string
	ExpirationTime time.Time

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

	GuestLinkID            string
	GuestLinkLabel         string
	GuestUploadMaxFileSize uint64
	GuestUploadCountLimit  int

	GuestLink struct {
		ID                   GuestLinkID
		Label                GuestLinkLabel
		Created              time.Time
		LastUsed             time.Time
		Expires              ExpirationTime
		UploadSizeRemaining  *GuestUploadMaxFileSize
		UploadCountRemaining *GuestUploadCountLimit
	}
)

// Treat a distant expiration time as sort of a sentinel value signifying a "never expire" option.
var NeverExpire = ExpirationTime(time.Date(2999, time.December, 31, 0, 0, 0, 0, time.UTC))

func (et ExpirationTime) String() string {
	return (time.Time(et)).String()
}
