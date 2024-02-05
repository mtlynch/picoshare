package checkers_test

import (
	"errors"
	"log"
	"math"
	"path/filepath"
	"testing"

	"github.com/mtlynch/picoshare/v2/space/checkers"
)

type mockFileSizer struct {
	size int64
}

func (s mockFileSizer) Size() int64 {
	return s.size
}

type mockFileSystem map[string]checkers.FileSizer

type mockFileSystemReader struct {
	fsStats    checkers.FileSystemStats
	fsStatsErr error
	fs         mockFileSystem
}

func (r mockFileSystemReader) GetFileSystemStats(path string) (checkers.FileSystemStats, error) {
	return r.fsStats, r.fsStatsErr
}

func (r mockFileSystemReader) FileSize(path string) (checkers.FileSizer, error) {
	log.Printf("getting file stats for %s", path) // DEBUG
	s, ok := r.fs[path]
	if !ok {
		return nil, errors.New("mock file not found")
	}
	return s, nil
}

func (r mockFileSystemReader) Glob(pattern string) ([]string, error) {
	log.Printf("searching glob patterns for %s", pattern) // DEBUG
	matches := []string{}
	for filename := range r.fs {
		match, err := filepath.Match(pattern, filename)
		if err != nil {
			return []string{}, err
		}
		if match {
			matches = append(matches, filename)
		}
	}
	log.Printf("matches: %v", matches) // DEBUG
	return matches, nil
}

func TestMeasureUsage(t *testing.T) {
	errDummyFsStats := errors.New("dummy fs stats error")
	for _, tt := range []struct {
		description   string
		fsStats       checkers.FileSystemStats
		fsStatsErr    error
		fs            mockFileSystem
		dbPath        string
		usageExpected checkers.PicoShareUsage
		errExpected   error
	}{
		{
			description: "aggregates filesystem stats",
			fsStats: checkers.FileSystemStats{
				FreeBlocks:  10,
				TotalBlocks: 30,
				BlockSize:   5,
			},
			fsStatsErr: nil,
			fs: mockFileSystem{
				"/dummy/store.db":      mockFileSizer{50},
				"/dummy/store.db-shm":  mockFileSizer{3},
				"/dummy/store.db-wal":  mockFileSizer{2},
				"/dummy/other-file.db": mockFileSizer{200},
			},
			dbPath: "/dummy/store.db",
			usageExpected: checkers.PicoShareUsage{
				PicoShareDbFileSize: 55,
				FileSystemUsage: checkers.FileSystemUsage{
					UsedBytes:  100,
					TotalBytes: 150,
				},
			},
			errExpected: nil,
		},
		{
			description:   "returns error when filesystem reader fails",
			fsStats:       checkers.FileSystemStats{},
			fsStatsErr:    errDummyFsStats,
			fs:            mockFileSystem{},
			dbPath:        "/dummy/store.db",
			usageExpected: checkers.PicoShareUsage{},
			errExpected:   errDummyFsStats,
		},
		{
			description: "returns error when filesystem claims files have negative sizes",
			fsStats: checkers.FileSystemStats{
				FreeBlocks:  10,
				TotalBlocks: 30,
				BlockSize:   5,
			},
			fsStatsErr: nil,
			fs: mockFileSystem{
				"/dummy/store.db":      mockFileSizer{50},
				"/dummy/store.db-shm":  mockFileSizer{-3},
				"/dummy/store.db-wal":  mockFileSizer{2},
				"/dummy/other-file.db": mockFileSizer{200},
			},
			dbPath: "/dummy/store.db",
			usageExpected: checkers.PicoShareUsage{
				PicoShareDbFileSize: 55,
				FileSystemUsage: checkers.FileSystemUsage{
					UsedBytes:  100,
					TotalBytes: 150,
				},
			},
			errExpected: checkers.ErrNegativeFileSize,
		},
		{
			description: "returns error when file sizes would cause integer overflow",
			fsStats: checkers.FileSystemStats{
				FreeBlocks:  10,
				TotalBlocks: 30,
				BlockSize:   5,
			},
			fsStatsErr: nil,
			fs: mockFileSystem{
				"/dummy/store.db":      mockFileSizer{math.MaxInt64},
				"/dummy/store.db-shm":  mockFileSizer{3},
				"/dummy/store.db-wal":  mockFileSizer{2},
				"/dummy/other-file.db": mockFileSizer{200},
			},
			dbPath: "/dummy/store.db",
			usageExpected: checkers.PicoShareUsage{
				PicoShareDbFileSize: 55,
				FileSystemUsage: checkers.FileSystemUsage{
					UsedBytes:  100,
					TotalBytes: 150,
				},
			},
			errExpected: checkers.ErrSizeOverflow,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			r := mockFileSystemReader{
				fsStats:    tt.fsStats,
				fsStatsErr: tt.fsStatsErr,
				fs:         tt.fs,
			}

			usage, err := checkers.NewFileSystemCheckerWithReader(tt.dbPath, r).MeasureUsage()
			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if err != nil {
				return
			}

			if got, want := usage, tt.usageExpected; got != want {
				t.Errorf("usage=%+v, want=%+v", got, want)
			}
		})
	}
}
