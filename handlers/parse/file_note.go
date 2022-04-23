package parse

import (
	"errors"

	"github.com/mtlynch/picoshare/v2/types"
)

// MaxFilenNoteLen is the maximum number of characters allowed in a file note.
const MaxFileNoteLen = 500

func FileNote(s string) (types.FileNote, error) {
	if s == "" {
		return nil, nil
	}
	if len(s) > MaxFileNoteLen {
		return nil, errors.New("filename too long")
	}
	// TODO: Check more rigorously
	return types.FileNote(&s), nil
}
