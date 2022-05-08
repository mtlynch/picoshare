package parse

import (
	"time"

	"github.com/mtlynch/picoshare/v2/types"
)

func ExpirationDate(s string) (types.ExpirationTime, error) {
	if s == "" {
		return types.NeverExpire, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return types.ExpirationTime{}, err
	}
	return types.ExpirationTime(t), nil
}
