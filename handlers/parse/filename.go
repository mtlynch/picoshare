package parse

import (
	"errors"
	"strings"

	"github.com/mtlynch/picoshare/v2/types"
)

// MaxFilenameLen is the maximum number of characters allowed for uploaded files
// There's no technical reason on PicoShare's side for this limitation, but it's
// useful to have some upper bound to limit malicious inputs, and 255 is a
// common filename limit across most filesystems.
const MaxFilenameLen = 255

func Filename(s string) (types.Filename, error) {
	if len(s) > MaxFilenameLen {
		return types.Filename(""), errors.New("filename too long")
	}
	if s == "." || strings.HasPrefix(s, "..") {
		return types.Filename(""), errors.New("illegal filename")
	}
	if strings.ContainsAny(s, "\\") {
		return types.Filename(""), errors.New("illegal characters in filename")
	}
	return types.Filename(s), nil
}
