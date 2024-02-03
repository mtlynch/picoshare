package space

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type (
	FileSystemChecker struct {
		dbPath string
	}

	FileSystemUsage struct {
		AvailableBytes uint64
		TotalBytes     uint64
	}

	PicoShareUsage struct {
		FileSystemUsage
		PicoShareDbFileSize uint64
	}
)

func NewFileSystemChecker(dbPath string) FileSystemChecker {
	return FileSystemChecker{dbPath}
}

func (fsc FileSystemChecker) MeasureUsage() (PicoShareUsage, error) {
	fsu, err := fsc.measureWholeFilesystem()
	if err != nil {
		return PicoShareUsage{}, nil
	}

	dbFilesSize, err := fsc.measureDbFileUsage()
	if err != nil {
		return PicoShareUsage{}, err
	}

	return PicoShareUsage{
		FileSystemUsage: FileSystemUsage{
			AvailableBytes: fsu.AvailableBytes,
			TotalBytes:     fsu.TotalBytes,
		},
		PicoShareDbFileSize: dbFilesSize,
	}, nil
}

func (fsc FileSystemChecker) measureWholeFilesystem() (FileSystemUsage, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(fsc.dbPath, &stat); err != nil {
		return FileSystemUsage{}, err
	}

	return FileSystemUsage{
		AvailableBytes: stat.Bfree * uint64(stat.Bsize),
		TotalBytes:     stat.Blocks * uint64(stat.Bsize),
	}, nil
}

func (fsc FileSystemChecker) measureDbFileUsage() (uint64, error) {
	// SQLite includes the .db file as well .db-shm and .db-wal.
	dbFilePattern := fsc.dbPath + "*"
	matches, err := filepath.Glob(dbFilePattern)
	if err != nil {
		return 0, err
	}

	var totalSize uint64
	for _, f := range matches {
		s, err := os.Stat(f)
		if err != nil {
			return 0, err
		}
		// Check negative.
		// Check overflow.
		totalSize += uint64(s.Size())
	}

	return totalSize, nil
}
