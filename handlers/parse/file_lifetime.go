package parse

import (
	"fmt"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

const minFileLifetimeInDays = 1
const maxFileLifetimeInYears = 10

// This is imprecise, but it's okay because file lifetimes are not exact
// measures of time.
const daysPerYear = 365

var (
	ErrFileLifetimeTooShort = fmt.Errorf("file lifetime must be at least %d days", minFileLifetimeInDays)
	ErrFileLifetimeTooLong  = fmt.Errorf("file lifetime must be at most %d years", maxFileLifetimeInYears)
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
