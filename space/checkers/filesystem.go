package checkers

import (
	"errors"
	"math/big"
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

	blockSize := big.NewInt(stat.Bsize)
	freeBlocks, err := uint64ToBigInt(stat.Bfree)
	if err != nil {
		return FileSystemUsage{}, err
	}
	totalBlocks, err := uint64ToBigInt(stat.Blocks)
	if err != nil {
		return FileSystemUsage{}, err
	}

	bAvailableBytes := big.NewInt(0).Mul(freeBlocks, blockSize)
	bTotalBytes := big.NewInt(0).Mul(totalBlocks, blockSize)
	bUsedBytes := big.NewInt(0).Sub(bTotalBytes, bAvailableBytes)

	usedBytes, err := bigIntToUint64(bUsedBytes)
	if err != nil {
		return FileSystemUsage{}, err
	}

	totalBytes, err := bigIntToUint64(bTotalBytes)
	if err != nil {
		return FileSystemUsage{}, err
	}

	return FileSystemUsage{
		UsedBytes:  usedBytes,
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

	totalSize := big.NewInt(0)
	for _, f := range matches {
		s, err := os.Stat(f)
		if err != nil {
			return 0, err
		}
		if s.Size() < 0 {
			return 0, errors.New("file size can't be negative")
		}
		bs := big.NewInt(s.Size())
		totalSize = totalSize.Add(totalSize, bs)
	}

	return bigIntToUint64(totalSize)
}
