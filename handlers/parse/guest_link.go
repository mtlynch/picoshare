package parse

import (
	"fmt"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

// Arbitrary limit to prevent too-long labels in the UI.
const MaxGuestLinkLabelLength = 200

var ErrGuestLinkLabelTooLong = fmt.Errorf("label too long - limit %d characters", MaxGuestLinkLabelLength)

func GuestLinkLabel(label string) (picoshare.GuestLinkLabel, error) {
	if len(label) > MaxGuestLinkLabelLength {
		return picoshare.GuestLinkLabel(""), ErrGuestLinkLabelTooLong
	}

	return picoshare.GuestLinkLabel(label), nil
}
