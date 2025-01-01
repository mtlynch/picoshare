package parse

import (
	"errors"
	"fmt"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

const expirationTimeFormat = time.RFC3339

var ErrExpirationUnrecognizedFormat = fmt.Errorf("unrecognized format for expiration time, must be in %s format", expirationTimeFormat)
var ErrExpirationTooSoon = errors.New("expire time must be at least one hour in the future")

func Expiration(expirationRaw string, now time.Time) (picoshare.ExpirationTime, error) {
	expiration, err := time.Parse(expirationTimeFormat, expirationRaw)
	if err != nil {
		return picoshare.ExpirationTime{}, ErrExpirationUnrecognizedFormat
	}

	if expiration.Sub(now) < (time.Hour * 1) {
		return picoshare.ExpirationTime{}, ErrExpirationTooSoon
	}

	return picoshare.ExpirationTime(expiration), nil
}
