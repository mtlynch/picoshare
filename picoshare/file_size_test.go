package picoshare_test

import (
	"math"
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestFileSizeFromInt(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   int
		want    uint64
		wantErr error
	}{
		{
			name:    "valid positive size",
			input:   100,
			want:    100,
			wantErr: nil,
		},
		{
			name:    "zero size",
			input:   0,
			want:    0,
			wantErr: picoshare.ErrEmptyFile,
		},
		{
			name:    "negative size",
			input:   -1,
			want:    0,
			wantErr: picoshare.ErrNegativeFileSize,
		},
		{
			name:    "maximum int value",
			input:   2147483647,
			want:    2147483647,
			wantErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := picoshare.FileSizeFromInt(tt.input)

			if got, want := err, tt.wantErr; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if err == nil {
				if got, want := fs.UInt64(), tt.want; got != want {
					t.Errorf("size=%v, want=%v", got, want)
				}
			}
		})
	}
}

func TestFileSizeFromUint64(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   uint64
		want    uint64
		wantErr error
	}{
		{
			name:    "valid positive size",
			input:   100,
			want:    100,
			wantErr: nil,
		},
		{
			name:    "zero size",
			input:   0,
			want:    0,
			wantErr: picoshare.ErrEmptyFile,
		},
		{
			name:    "maximum uint64 value",
			input:   math.MaxUint64,
			want:    math.MaxUint64,
			wantErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := picoshare.FileSizeFromUint64(tt.input)

			if got, want := err, tt.wantErr; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if err != nil {
				return
			}

			if got, want := fs.UInt64(), tt.want; got != want {
				t.Errorf("size=%v, want=%v", got, want)
			}
		})
	}
}

func TestFileSizeEquality(t *testing.T) {
	for _, tt := range []struct {
		name      string
		size1     uint64
		size2     uint64
		wantEqual bool
	}{
		{
			name:      "equal sizes",
			size1:     100,
			size2:     100,
			wantEqual: true,
		},
		{
			name:      "different sizes",
			size1:     100,
			size2:     15,
			wantEqual: false,
		},
		{
			name:      "equal large sizes",
			size1:     math.MaxUint64,
			size2:     math.MaxUint64,
			wantEqual: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fs1, err := picoshare.FileSizeFromUint64(tt.size1)
			if err != nil {
				t.Fatalf("failed to create first FileSize: %v", err)
			}

			fs2, err := picoshare.FileSizeFromUint64(tt.size2)
			if err != nil {
				t.Fatalf("failed to create second FileSize: %v", err)
			}

			if got, want := fs1.Equal(fs2), tt.wantEqual; got != want {
				t.Errorf("Equal()=%v, want=%v", got, want)
			}
		})
	}
}
