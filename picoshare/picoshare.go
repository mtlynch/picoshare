package picoshare

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type (
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
		// DownloadPassphrase is empty when downloads do not require a passphrase.
		//
		// PicoShare intentionally stores download passphrases in plaintext instead
		// of hashing them. The server owner assigns every download passphrase, so
		// they are not user credentials that a hash would protect, and hashing
		// would stop the owner from reading or changing a passphrase later.
		DownloadPassphrase DownloadPassphrase
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

// entryIDCharacters omits visually similar characters (I, l, 1), (0, O).
const (
	entryIDLength     = 10
	entryIDCharacters = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// EntryID identifies an uploaded entry.
type EntryID struct {
	value string
}

// Treat a distant expiration time as sort of a sentinel value signifying a "never expire" option.
var NeverExpire = ExpirationTime(time.Date(2999, time.December, 31, 0, 0, 0, 0, time.UTC))

func (id EntryID) String() string {
	return id.value
}

// NewEntryID constructs an entry ID from user-provided text.
func NewEntryID(raw string) (EntryID, error) {
	if len(raw) != entryIDLength {
		return EntryID{}, fmt.Errorf(
			"entry ID has invalid length: got %d, want %d", len(raw), entryIDLength)
	}

	for _, character := range raw {
		if !strings.ContainsRune(entryIDCharacters, character) {
			return EntryID{}, fmt.Errorf("entry ID contains invalid character: %q", character)
		}
	}

	return EntryID{value: raw}, nil
}

// MustCreateEntryID constructs an entry ID or panics when raw is invalid.
func MustCreateEntryID(raw string) EntryID {
	id, err := NewEntryID(raw)
	if err != nil {
		panic(fmt.Sprintf("failed to create entry ID: %v", err))
	}
	return id
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
