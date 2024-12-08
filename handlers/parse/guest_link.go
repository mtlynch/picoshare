package parse

import (
	"errors"
	"fmt"
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

// Arbitrary limit to prevent too-long labels in the UI.
const MaxGuestLinkLabelLength = 200
const validTimeUnits = "1ns, 1us (or 1µs), 1ms, 1s, 1m, 1h"

var ErrGuestLinkLabelTooLong = fmt.Errorf("label too long - limit %d characters", MaxGuestLinkLabelLength)
var ErrFileLifeTimeUnrecognizedFormat = fmt.Errorf("unrecognized format for file life time, must be in %s format", validTimeUnits)
var ErrFileLifeTimeTooShort = errors.New("file life time must be at least one hour in the future")

func GuestLinkLabel(label string) (picoshare.GuestLinkLabel, error) {
	if len(label) > MaxGuestLinkLabelLength {
		return picoshare.GuestLinkLabel(""), ErrGuestLinkLabelTooLong
	}

	return picoshare.GuestLinkLabel(label), nil
}

func GuestFileLifeTime(fileLifetimeRaw string) (picoshare.FileLifetime, error) {
	dur, err := time.ParseDuration(fileLifetimeRaw)
	if err != nil {
		return picoshare.FileLifetime{}, ErrFileLifeTimeUnrecognizedFormat
	}

	if picoshare.NewFileLifetimeFromDuration(dur) == picoshare.FileLifetimeInfinite {
		return picoshare.FileLifetimeInfinite, nil
	}

	if dur < (time.Hour * 1) {
		return picoshare.FileLifetime{}, ErrFileLifeTimeTooShort
	}

	return picoshare.NewFileLifetimeFromDuration(dur), nil
}
