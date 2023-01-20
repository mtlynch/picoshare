package parse

import (
	"errors"
	"regexp"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

// MaxFileNoteBytes is the maximum number of bytes allowed in a file note.
const MaxFileNoteBytes = 500

// illegalNoteTagPattern matches tags we don't allow in file notes. We have
// other protections in place to prevent XSS and escaping HTML encoding, but
// this is just defense in depth to catch highly suspicious strings that we
// don't want to risk mishandling later.
var illegalNoteTagPattern = regexp.MustCompile(`<\s*/?((script)|(iframe))\s*>`)

func FileNote(s string) (picoshare.FileNote, error) {
	if s == "" {
		return picoshare.FileNote{}, nil
	}
	if len(s) > MaxFileNoteBytes {
		return picoshare.FileNote{}, errors.New("note is too long")
	}
	if err := checkJavaScriptNullOrUndefined(s); err != nil {
		return picoshare.FileNote{}, err
	}
	if illegalNoteTagPattern.MatchString(s) {
		return picoshare.FileNote{}, errors.New("note must not contain HTML tags")
	}
	return picoshare.FileNote{Value: &s}, nil
}

// If the client sent a value of 'null' or 'undefined', it's likely a JS error
// and not literally what the end-user submitted, so reject it.
func checkJavaScriptNullOrUndefined(s string) error {
	if s == "null" {
		return errors.New("value of 'null' is not allowed")
	}
	if s == "undefined" {
		return errors.New("value of 'undefined' is not allowed")
	}
	return nil
}
