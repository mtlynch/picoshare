package picoshare

import (
	"fmt"
	"io"
	"time"

	"github.com/mtlynch/picoshare/random"
)

// picoshareEpoch is the first release date, matching migration 015's lower
// bound for timestamps.
var picoshareEpoch = time.Date(2022, time.February, 20, 0, 0, 0, 0, time.UTC)

// NeverExpire represents the "never expire" option with a distant timestamp.
var NeverExpire = ExpirationTime{value: time.Date(2999, time.December, 31, 0, 0, 0, 0, time.UTC)}

// ExpirationTime is an expiration timestamp at or after PicoShare's first
// release. The zero value is invalid; use NewExpirationTime to construct one.
type ExpirationTime struct {
	value time.Time
}

func (et ExpirationTime) String() string {
	return et.Time().String()
}

func (et ExpirationTime) Time() time.Time {
	return et.value
}

// NewExpirationTime constructs an expiration timestamp, including past
// expirations, but rejects timestamps before PicoShare's first release.
func NewExpirationTime(raw time.Time) (ExpirationTime, error) {
	if raw.Before(picoshareEpoch) {
		return ExpirationTime{}, fmt.Errorf("expiration time must be at or after %s", picoshareEpoch.Format(time.RFC3339))
	}
	return ExpirationTime{value: raw}, nil
}

type (
	Filename    string
	ContentType string

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

// EntryIDLength is the number of characters in an entry ID.
const EntryIDLength = 10

// entryIDCharacters omits visually similar characters (I, l, 1), (0, O).
const entryIDCharacters = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var entryIDCharacterSet = func() map[rune]struct{} {
	characters := make(map[rune]struct{}, len(entryIDCharacters))
	for _, character := range entryIDCharacters {
		characters[character] = struct{}{}
	}
	return characters
}()

// EntryID identifies an uploaded entry.
type EntryID struct {
	value string
}

func (id EntryID) String() string {
	return id.value
}

// EntryIDFromString constructs an entry ID from user-provided text.
func EntryIDFromString(raw string) (EntryID, error) {
	if len(raw) != EntryIDLength {
		return EntryID{}, fmt.Errorf(
			"entry ID has invalid length: got %d, want %d", len(raw), EntryIDLength)
	}

	for _, character := range raw {
		if _, ok := entryIDCharacterSet[character]; !ok {
			return EntryID{}, fmt.Errorf("entry ID contains invalid character: %q", character)
		}
	}

	return EntryID{value: raw}, nil
}

// NewEntryID generates an entry ID.
func NewEntryID() EntryID {
	return EntryID{value: random.String(EntryIDLength, []rune(entryIDCharacters))}
}

// MustCreateEntryID constructs an entry ID or panics when raw is invalid.
func MustCreateEntryID(raw string) EntryID {
	id, err := EntryIDFromString(raw)
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

func (n FileNote) String() string {
	if n.Value == nil {
		return "<nil>"
	}
	return *n.Value
}
