package space

import "github.com/mtlynch/picoshare/v2/picoshare"

type (
	DatabaseMetadataReader interface {
		GetEntriesMetadata() ([]picoshare.UploadMetadata, error)
	}

	DatabaseChecker struct {
		reader DatabaseMetadataReader
	}
)

func NewDatabaseChecker(dbReader DatabaseMetadataReader) DatabaseChecker {
	return DatabaseChecker{dbReader}
}

func (dbc DatabaseChecker) TotalSize() (uint64, error) {
	var dbTotal uint64
	entries, err := dbc.reader.GetEntriesMetadata()
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		// TODO: Check for overflow.
		dbTotal += entry.Size
	}

	return dbTotal, nil
}
