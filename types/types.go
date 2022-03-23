package types

import (
	"io"
	"net"
	"time"
)

type EntryID string
type Filename string
type ContentType string
type ExpirationTime time.Time

type UploadMetadata struct {
	ID          EntryID
	Filename    Filename
	ContentType ContentType
	Uploaded    time.Time
	Expires     ExpirationTime
	UploaderIP  net.IP
	Size        int
}

type UploadEntry struct {
	UploadMetadata
	Reader io.ReadSeeker
}
