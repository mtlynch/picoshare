package space

import (
	"golang.org/x/sys/unix"
)

type (
	Checker interface {
		Check() (CheckResult, error)
	}

	defaultChecker struct {
		dataDir string
	}

	CheckResult struct {
		AvailableBytes uint64
		TotalBytes     uint64
	}
)

func NewChecker(dir string) Checker {
	return defaultChecker{dir}
}

func (c defaultChecker) Check() (CheckResult, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(c.dataDir, &stat); err != nil {
		return CheckResult{}, err
	}

	return CheckResult{
		AvailableBytes: stat.Bfree * uint64(stat.Bsize),
		TotalBytes:     stat.Blocks * uint64(stat.Bsize),
	}, nil
}
