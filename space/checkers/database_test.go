package checkers_test

import (
	"errors"
	"math"
	"testing"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/space/checkers"
)

type mockDatabaseReader struct {
	metadataEntries []picoshare.UploadMetadata
	err             error
}

func (r mockDatabaseReader) GetEntriesMetadata() ([]picoshare.UploadMetadata, error) {
	return r.metadataEntries, r.err
}

func TestTotalSize(t *testing.T) {
	dummyDatabaseReaderErr := errors.New("dummy database reader error")
	for _, tt := range []struct {
		description   string
		dbEntries     []picoshare.UploadMetadata
		dbErr         error
		totalExpected uint64
		errExpected   error
	}{
		{
			description: "returns the sum of the database entries",
			dbEntries: []picoshare.UploadMetadata{
				{
					Size: 5,
				},
				{
					Size: 3,
				},
				{
					Size: 1,
				},
			},
			dbErr:         nil,
			totalExpected: 9,
			errExpected:   nil,
		},
		{
			description: "returns an error if the database sizes overflow int64",
			dbEntries: []picoshare.UploadMetadata{
				{
					Size: math.MaxUint64,
				},
				{
					Size: 1,
				},
			},
			dbErr:         nil,
			totalExpected: 0,
			errExpected:   checkers.ErrSizeOverflow,
		},
		{
			description:   "returns error when database checker fails",
			dbEntries:     []picoshare.UploadMetadata{},
			dbErr:         dummyDatabaseReaderErr,
			totalExpected: 0,
			errExpected:   dummyDatabaseReaderErr,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			r := mockDatabaseReader{
				metadataEntries: tt.dbEntries,
				err:             tt.dbErr,
			}

			total, err := checkers.NewDatabaseChecker(r).TotalSize()
			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if got, want := total, tt.totalExpected; got != want {
				t.Errorf("total=%+v, want=%+v", got, want)
			}
		})
	}
}
