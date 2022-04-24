package parse

import (
	"errors"
	"regexp"

	"github.com/mtlynch/picoshare/v2/types"
)

// MaxFilenNoteLen is the maximum number of characters allowed in a file note.
const MaxFileNoteLen = 500

// illegalNoteTagPattern matches tags we don't allow in file notes. We have
// other protections in place to prevent XSS and escaping HTML encoding, but
// this is just defense in depth to catch highly suspicious strings that we
// don't want to risk mishandling later.
var illegalNoteTagPattern = regexp.MustCompile(`<\s*/?((script)|(iframe))\s*>`)

func FileNote(s string) (types.FileNote, error) {
	if s == "" {
		return nil, nil
	}
	if len(s) > MaxFileNoteLen {
		return nil, errors.New("note is too long")
	}
	if err := checkJavaScriptNullOrUndefined(s); err != nil {
		return nil, err
	}
	if illegalNoteTagPattern.MatchString(s) {
		return nil, errors.New("note must not contain HTML tags")
	}
	return types.FileNote(&s), nil
}

func checkJavaScriptNullOrUndefined(s string) error {
	if s == "null" {
		return errors.New("value of 'null' is not allowed")
	}
	if s == "undefined" {
		return errors.New("value of 'undefined' is not allowed")
	}
	return nil
}
