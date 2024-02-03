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
		UsedBytes  uint64
		TotalBytes uint64
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
			UsedBytes:  fsu.UsedBytes,
			TotalBytes: fsu.TotalBytes,
		},
		PicoShareDbFileSize: dbFilesSize,
	}, nil
}

func (fsc FileSystemChecker) measureWholeFilesystem() (FileSystemUsage, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(fsc.dbPath, &stat); err != nil {
		return FileSystemUsage{}, err
	}
	availableBytes := stat.Bfree * uint64(stat.Bsize)
	totalBytes := stat.Blocks * uint64(stat.Bsize)

	return FileSystemUsage{
		UsedBytes:  totalBytes - availableBytes,
		TotalBytes: totalBytes,
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
