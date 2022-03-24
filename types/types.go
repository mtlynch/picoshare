package types

import (
	"io"
	"time"
)

type EntryID string
type Filename string
type ContentType string
type ExpirationTime time.Time
type Filesize int

type UploadMetadata struct {
	ID          EntryID
	Filename    Filename
	ContentType ContentType
	Uploaded    time.Time
	Expires     ExpirationTime
	Size        Filesize
}

type UploadEntry struct {
	UploadMetadata
	Reader io.ReadSeeker
}
