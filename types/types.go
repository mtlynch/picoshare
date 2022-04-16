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
		GuestLinkID GuestLinkID
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

	GuestLinkID             string
	GuestLinkLabel          string
	GuestUploadMaxFileBytes uint64
	GuestUploadCountLimit   int

	GuestLink struct {
		ID                   GuestLinkID
		Label                GuestLinkLabel
		Created              time.Time
		Expires              ExpirationTime
		MaxFileBytes         *GuestUploadMaxFileBytes
		UploadCountRemaining *GuestUploadCountLimit
	}
)

// Treat a distant expiration time as sort of a sentinel value signifying a "never expire" option.
var NeverExpire = ExpirationTime(time.Date(2999, time.December, 31, 0, 0, 0, 0, time.UTC))

func (et ExpirationTime) String() string {
	return (time.Time(et)).String()
}

func (gl GuestLink) CanAcceptMoreFiles() bool {
	if gl.UploadCountRemaining == nil {
		return true
	}
	r := int(*gl.UploadCountRemaining)
	return r > 0
}

func (gl *GuestLink) DecrementUploadCount() {
	if gl.UploadCountRemaining == nil {
		return
	}
	s := GuestUploadCountLimit(int(*gl.UploadCountRemaining) - 1)
	gl.UploadCountRemaining = &s
}
