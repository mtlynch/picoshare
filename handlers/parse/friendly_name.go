package parse

import (
	"errors"
	"regexp"

	"github.com/mtlynch/picoshare/picoshare"
)

const MaxFriendlyNameBytes = 200

var ErrFriendlyNameEmpty = errors.New("friendly name must be non-empty")
var ErrFriendlyNameTooLong = errors.New("friendly name too long")
var ErrFriendlyNameInvalidCharacters = errors.New("friendly name contains invalid characters")

var friendlyNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\._-]+$`)

func FriendlyName(s string) (picoshare.FriendlyName, error) {
	if s == "" {
		return picoshare.FriendlyName(""), ErrFriendlyNameEmpty
	}
	if len(s) > MaxFriendlyNameBytes {
		return picoshare.FriendlyName(""), ErrFriendlyNameTooLong
	}
	if !friendlyNameRegex.MatchString(s) {
		return picoshare.FriendlyName(""), ErrFriendlyNameInvalidCharacters
	}
	return picoshare.FriendlyName(s), nil
}
