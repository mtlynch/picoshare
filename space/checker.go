package space

import "github.com/mtlynch/picoshare/v2/space/checkers"

type (
	FileSystemChecker interface {
		MeasureUsage() (checkers.PicoShareUsage, error)
	}

	DatabaseChecker interface {
		TotalSize() (uint64, error)
	}

	Checker struct {
		fsChecker FileSystemChecker
		dbChecker DatabaseChecker
	}

	Usage struct {
		DataSize         uint64
		DatabaseFileSize uint64
		UsedBytes        uint64
		TotalBytes       uint64
	}
)

func NewChecker(dbPath string, dbReader checkers.DatabaseMetadataReader) Checker {
	return NewCheckerFromCheckers(checkers.NewFileSystemChecker(dbPath), checkers.NewDatabaseChecker(dbReader))
}

func NewCheckerFromCheckers(fsChecker FileSystemChecker, dbChecker DatabaseChecker) Checker {
	return Checker{
		fsChecker,
		dbChecker,
	}
}

func (c Checker) Check() (Usage, error) {
	fsUsage, err := c.fsChecker.MeasureUsage()
	if err != nil {
		return Usage{}, err
	}

	dbTotalSize, err := c.dbChecker.TotalSize()
	if err != nil {
		return Usage{}, err
	}

	return Usage{
		DataSize:         dbTotalSize,
		DatabaseFileSize: fsUsage.PicoShareDbFileSize,
		UsedBytes:        fsUsage.UsedBytes,
		TotalBytes:       fsUsage.TotalBytes,
	}, nil
}
