package parse

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

// Arbitrary limit to prevent too-long labels in the UI.
const MaxGuestLinkLabelLength = 200
const validTimeUnits = "1ns, 1us (or 1µs), 1ms, 1s, 1m, 1h"

var ErrGuestLinkLabelTooLong = fmt.Errorf("label too long - limit %d characters", MaxGuestLinkLabelLength)
var ErrFileLifeTimeUnrecognizedFormat = fmt.Errorf("unrecognized format for file life time, must be in %s format", validTimeUnits)
var ErrFileLifeTimeTooSoon = errors.New("file life time must be at least one hour in the future")

func GuestLinkLabel(label string) (picoshare.GuestLinkLabel, error) {
	if len(label) > MaxGuestLinkLabelLength {
		return picoshare.GuestLinkLabel(""), ErrGuestLinkLabelTooLong
	}

	return picoshare.GuestLinkLabel(label), nil
}

func GuestFileLifeTime(fileLifeTimeRaw string) (picoshare.FileLifetime, error) {
	t, err := time.Parse(expirationTimeFormat, fileLifeTimeRaw)
	if err != nil {
		return picoshare.FileLifetime{}, ErrExpirationUnrecognizedFormat
	}

	if picoshare.ExpirationTime(t) == picoshare.NeverExpire {
		return picoshare.FileLifetimeInfinite, nil
	}

	delta := time.Until(time.Time(t))
	fileLifeTime := fmt.Sprintf("%.0fh", math.Round(delta.Hours()))
	expiration, err := time.ParseDuration(fileLifeTime)
	if err != nil {
		return picoshare.FileLifetime{}, ErrFileLifeTimeUnrecognizedFormat
	}

	if expiration < (time.Hour * 1) {
		return picoshare.FileLifetime{}, ErrFileLifeTimeTooSoon
	}

	return picoshare.NewFileLifetimeFromDuration(expiration), nil
}
