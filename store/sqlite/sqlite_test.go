package sqlite_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/vfs/memdb"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/sqlite"
	"github.com/mtlynch/picoshare/testutil"
)

func TestNew(t *testing.T) {
	for _, tt := range []struct {
		explanation           string
		optimizeForLitestream bool
	}{
		{
			explanation: "opening an existing database preserves uploads",
		},
		{
			explanation:           "opening an existing database with Litestream settings preserves uploads",
			optimizeForLitestream: true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			params := sqlite.Params{
				Path:                  memdb.TestDB(t),
				OptimizeForLitestream: tt.optimizeForLitestream,
				Now: func() time.Time {
					return time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
				},
			}
			original := sqlite.New(params)
			data := "hello, world!"
			metadata := picoshare.UploadMetadata{
				ID:       picoshare.MustCreateEntryID("abcdefghij"),
				Filename: "original.txt",
				Uploaded: mustParseTime("2025-05-25T00:00:00Z"),
				Expires:  testutil.MustParseExpirationTime(t, "2040-01-01T00:00:00Z"),
				Size:     mustParseFileSize(len(data)),
			}
			if err := original.InsertEntry(strings.NewReader(data), metadata); err != nil {
				t.Fatalf("failed to insert upload: %v", err)
			}

			reopened := sqlite.New(params)
			stored, err := reopened.GetEntryMetadata(metadata.ID)
			if err != nil {
				t.Fatalf("failed to read existing metadata: %v", err)
			}
			if got, want := stored.Filename, metadata.Filename; got != want {
				t.Errorf("filename=%q, want=%q", got, want)
			}
			if got, want := stored.Uploaded, metadata.Uploaded; !got.Equal(want) {
				t.Errorf("upload time=%v, want=%v", got, want)
			}
			if got, want := stored.Expires, metadata.Expires; got != want {
				t.Errorf("expiration=%v, want=%v", got, want)
			}
			reader, err := reopened.ReadEntryFile(metadata.ID)
			if err != nil {
				t.Fatalf("failed to open existing upload: %v", err)
			}
			contents, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read existing upload: %v", err)
			}
			if got, want := string(contents), data; got != want {
				t.Errorf("contents=%q, want=%q", got, want)
			}
		})
	}
}
