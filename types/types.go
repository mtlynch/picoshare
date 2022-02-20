package types

import "time"

type EntryID string
type Filename string

type UploadMetadata struct {
	ID       EntryID
	Filename Filename
	Uploaded time.Time
	Expires  time.Time
	Size     int
}

type UploadEntry struct {
	UploadMetadata
	Data []byte
}
