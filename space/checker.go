package space

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type (
	DatabaseTotaler interface {
		TotalSize() (uint64, error)
	}

	Checker struct {
		dbPath    string
		dbTotaler DatabaseTotaler
	}

	CheckResult struct {
		DataSize       uint64
		DbSize         uint64
		AvailableBytes uint64
		TotalBytes     uint64
	}
)

func NewChecker(dbPath string, dbTotaler DatabaseTotaler) Checker {
	return Checker{
		dbPath:    dbPath,
		dbTotaler: dbTotaler,
	}
}

func (c Checker) Check() (CheckResult, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(c.dbPath, &stat); err != nil {
		return CheckResult{}, err
	}

	matches, err := filepath.Glob(c.dbPath + "*")
	if err != nil {
		return CheckResult{}, err
	}

	var totalSize uint64
	for _, f := range matches {
		s, err := os.Stat(f)
		if err != nil {
			return CheckResult{}, err
		}
		totalSize += uint64(s.Size())
	}

	dbTotalSize, err := c.dbTotaler.TotalSize()
	if err != nil {
		return CheckResult{}, err
	}

	return CheckResult{
		DataSize:       totalSize,
		DbSize:         dbTotalSize,
		AvailableBytes: stat.Bfree * uint64(stat.Bsize),
		TotalBytes:     stat.Blocks * uint64(stat.Bsize),
	}, nil
}
