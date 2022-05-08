package parse

import (
	"errors"
	"fmt"
	"time"

	"github.com/mtlynch/picoshare/v2/types"
)

const expirationTimeFormat = time.RFC3339

var ErrExpirationUnrecognizedFormat = fmt.Errorf("unrecognized format for expiration time, must be in %s format", expirationTimeFormat)
var ErrExpirationTooSoon = errors.New("expire time must be at least one hour in the future")

func Expiration(expirationRaw string) (types.ExpirationTime, error) {
	expiration, err := time.Parse(expirationTimeFormat, expirationRaw)
	if err != nil {
		return types.ExpirationTime{}, ErrExpirationUnrecognizedFormat
	}

	if time.Until(expiration) < (time.Hour * 1) {
		return types.ExpirationTime{}, ErrExpirationTooSoon
	}

	return types.ExpirationTime(expiration), nil
}
