package parse

import (
	"errors"
	"strings"

	"github.com/mtlynch/picoshare/v2/types"
)

// MaxFilenameBytes is the maximum number of bytes allowed for uploaded files
// There's no technical reason on PicoShare's side for this limitation, but it's
// useful to have some upper bound to limit malicious inputs, and 255 is a
// common filename limit (in single-byte characters) across most filesystems.
const MaxFilenameBytes = 255

var ErrFilenameEmpty = errors.New("filename must be non-empty")
var ErrFilenameTooLong = errors.New("filename too long")
var ErrFilenameHasDotPrefix = errors.New("filename cannot begin with dots")
var ErrFilenameIllegalCharacters = errors.New("illegal characters in filename")

func Filename(s string) (types.Filename, error) {
	if s == "" {
		return types.Filename(""), ErrFilenameEmpty
	}
	if len(s) > MaxFilenameBytes {
		return types.Filename(""), ErrFilenameTooLong
	}
	if s == "." || strings.HasPrefix(s, "..") {
		return types.Filename(""), ErrFilenameHasDotPrefix
	}
	if strings.ContainsAny(s, "\\") {
		return types.Filename(""), ErrFilenameIllegalCharacters
	}
	return types.Filename(s), nil
}
