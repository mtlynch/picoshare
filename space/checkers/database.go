package checkers

import (
	"errors"
	"math"
	"math/big"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

type (
	DatabaseMetadataReader interface {
		GetEntriesMetadata() ([]picoshare.UploadMetadata, error)
	}

	DatabaseChecker struct {
		reader DatabaseMetadataReader
	}
)

var ErrSizeOverflow = errors.New("values would cause an overflow")

func NewDatabaseChecker(dbReader DatabaseMetadataReader) DatabaseChecker {
	return DatabaseChecker{dbReader}
}

func (dbc DatabaseChecker) TotalSize() (uint64, error) {
	dbTotal := big.NewInt(0)
	entries, err := dbc.reader.GetEntriesMetadata()
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		bigSize, err := uint64ToBigInt(entry.Size)
		if err != nil {
			return 0, err
		}
		dbTotal = dbTotal.Add(dbTotal, bigSize)
	}

	return bigIntToUint64(dbTotal)
}

func uint64ToBigInt(val uint64) (*big.Int, error) {
	if val > math.MaxInt64 {
		return big.NewInt(0), ErrSizeOverflow
	}
	return big.NewInt(int64(val)), nil
}

func bigIntToUint64(b *big.Int) (uint64, error) {
	// We could create a big.Int that's equal to MaxUint64, but it's probably not
	// necessary in practice.
	if b.Cmp(big.NewInt(math.MaxInt64)) > 0 {
		return 0, ErrSizeOverflow
	}

	return b.Uint64(), nil
}
