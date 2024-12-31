package parse

import (
	"fmt"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

const minFileLifetimeInDays = 1
const maxFileLifetimeInYears = 10

// This is imprecise, but it's okay because file lifetimes are not exact
// measures of time.
const daysPerYear = 365

var (
	ErrFileLifetimeTooShort           = fmt.Errorf("file lifetime must be at least %d days", minFileLifetimeInDays)
	ErrFileLifetimeTooLong            = fmt.Errorf("file lifetime must be at most %d years", maxFileLifetimeInYears)
	ErrFileLifetimeUnrecognizedFormat = fmt.Errorf("unrecognized format for file life time, must be in 1ns, 1us (or 1µs), 1ms, 1s, 1m, 1h format")
)

func FileLifetime(lifetimeInDays uint16) (picoshare.FileLifetime, error) {
	if lifetimeInDays < minFileLifetimeInDays {
		return picoshare.FileLifetime{}, ErrFileLifetimeTooShort
	}
	if lifetimeInDays > (maxFileLifetimeInYears * daysPerYear) {
		return picoshare.FileLifetime{}, ErrFileLifetimeTooLong
	}
	return picoshare.NewFileLifetimeInDays(lifetimeInDays), nil
}

func FileLifetimeFromString(fileLifetimeRaw string) (picoshare.FileLifetime, error) {
	dur, err := time.ParseDuration(fileLifetimeRaw)
	if err != nil {
		return picoshare.FileLifetime{}, ErrFileLifetimeUnrecognizedFormat
	}

	if flt, _ := picoshare.NewFileLifetimeFromDuration(dur); flt == picoshare.FileLifetimeInfinite {
		return picoshare.FileLifetimeInfinite, nil
	}

	return picoshare.NewFileLifetimeFromDuration(dur)
}
