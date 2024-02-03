package space

type (
	Checker struct {
		fsChecker FileSystemChecker
		dbChecker DatabaseChecker
	}

	CheckResult struct {
		DataSize         uint64
		DatabaseFileSize uint64
		UsedBytes        uint64
		TotalBytes       uint64
	}
)

func NewChecker(dbPath string, dbReader DatabaseMetadataReader) Checker {
	return NewCheckerFromCheckers(NewFileSystemChecker(dbPath), NewDatabaseChecker(dbReader))
}

func NewCheckerFromCheckers(fsChecker FileSystemChecker, dbChecker DatabaseChecker) Checker {
	return Checker{
		fsChecker,
		dbChecker,
	}
}

func (c Checker) Check() (CheckResult, error) {
	fsUsage, err := c.fsChecker.MeasureUsage()
	if err != nil {
		return CheckResult{}, err
	}

	dbTotalSize, err := c.dbChecker.TotalSize()
	if err != nil {
		return CheckResult{}, err
	}

	return CheckResult{
		DataSize:         dbTotalSize,
		DatabaseFileSize: fsUsage.PicoShareDbFileSize,
		UsedBytes:        fsUsage.TotalBytes - fsUsage.AvailableBytes,
		TotalBytes:       fsUsage.TotalBytes,
	}, nil
}
