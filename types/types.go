package types

import "time"

type EntryID string
type Filename string

type UploadEntry struct {
	Filename Filename
	Uploaded time.Time
	Expires  time.Time
	Data     []byte
}
