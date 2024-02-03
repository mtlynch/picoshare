package space

import (
	"os"
	"path/filepath"

	"github.com/mtlynch/picoshare/v2/picoshare"
	"golang.org/x/sys/unix"
)

type (
	DatabaseMetadataReader interface {
		GetEntriesMetadata() ([]picoshare.UploadMetadata, error)
	}

	Checker struct {
		dbPath   string
		dbReader DatabaseMetadataReader
	}

	CheckResult struct {
		DataSize       uint64
		DbSize         uint64
		AvailableBytes uint64
		TotalBytes     uint64
	}
)

func NewChecker(dbPath string, dbReader DatabaseMetadataReader) Checker {
	return Checker{
		dbPath:   dbPath,
		dbReader: dbReader,
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

	var dbTotal uint64
	entries, err := c.dbReader.GetEntriesMetadata()
	if err != nil {
		return CheckResult{}, err
	}

	for _, entry := range entries {
		dbTotal += entry.Size
	}

	return CheckResult{
		DataSize:       totalSize,
		DbSize:         dbTotal,
		AvailableBytes: stat.Bfree * uint64(stat.Bsize),
		TotalBytes:     stat.Blocks * uint64(stat.Bsize),
	}, nil
}
