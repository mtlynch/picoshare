package picoshare

import (
	"io"
	"time"

	"github.com/mtlynch/picoshare/v2/kdf"
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
		ID            EntryID
		Filename      Filename
		Note          FileNote
		ContentType   ContentType
		Uploaded      time.Time
		Expires       ExpirationTime
		Size          FileSize
		GuestLink     GuestLink
		DownloadCount uint64
		// PassphraseKey stores a derived key if the uploader protected the file
		// with a passphrase. Zero-value means no protection.
		PassphraseKey kdf.DerivedKey
	}

	DownloadRecord struct {
		Time      time.Time
		ClientIP  string
		UserAgent string
	}

	UploadEntry struct {
		UploadMetadata
		Reader io.ReadSeeker
	}
)

// Treat a distant expiration time as sort of a sentinel value signifying a "never expire" option.
var NeverExpire = ExpirationTime(time.Date(2999, time.December, 31, 0, 0, 0, 0, time.UTC))

func (id EntryID) String() string {
	return string(id)
}

func (f Filename) String() string {
	return string(f)
}

func (ct ContentType) String() string {
	return string(ct)
}

func (et ExpirationTime) String() string {
	return et.Time().String()
}

func (et ExpirationTime) Time() time.Time {
	return time.Time(et)
}

func (n FileNote) String() string {
	if n.Value == nil {
		return "<nil>"
	}
	return *n.Value
}
