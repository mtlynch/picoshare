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

	FileNote struct {
		Value *string
	}

	UploadMetadata struct {
		ID          EntryID
		GuestLinkID GuestLinkID
		Filename    Filename
		Note        FileNote
		ContentType ContentType
		Uploaded    time.Time
		Expires     ExpirationTime
		Size        int64
	}

	UploadEntry struct {
		UploadMetadata
		Reader io.ReadSeeker
	}
)

// Treat a distant expiration time as sort of a sentinel value signifying a "never expire" option.
var NeverExpire = ExpirationTime(time.Date(2999, time.December, 31, 0, 0, 0, 0, time.UTC))

func (et ExpirationTime) String() string {
	return (time.Time(et)).String()
}

func (n FileNote) String() string {
	if n.Value == nil {
		return "<nil>"
	}
	return *n.Value
}
