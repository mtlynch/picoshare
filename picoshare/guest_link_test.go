package picoshare_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/testutil"
)

func TestGuestLinkIsActive(t *testing.T) {
	for _, tt := range []struct {
		explanation  string
		now          time.Time
		guestLink    picoshare.GuestLink
		isActiveWant bool
	}{
		{
			explanation: "a link whose expiration is after the current time should be marked as active",
			now:         time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires: testutil.MustCreate(t, picoshare.NewExpirationTime, time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)),
			},
			isActiveWant: true,
		},
		{
			explanation: "a link whose expiration equals the current time should be marked as inactive",
			now:         time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires: testutil.MustCreate(t, picoshare.NewExpirationTime, time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)),
			},
			isActiveWant: false,
		},
		{
			explanation: "a link whose expiration is before the current time should be marked as inactive",
			now:         time.Date(2025, time.January, 3, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires: testutil.MustCreate(t, picoshare.NewExpirationTime, time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)),
			},
			isActiveWant: false,
		},
		{
			explanation: "an enabled link with neither an expiration time nor an upload count limit should be marked as active",
			now:         time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires: picoshare.NeverExpire,
			},
			isActiveWant: true,
		},
		{
			explanation: "a disabled link should be marked as inactive even if it has neither an expiration time nor an upload count limit",
			now:         time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires: picoshare.NeverExpire,
				IsDisabled: true,
			},
			isActiveWant: false,
		},
		{
			explanation: "a link whose upload count equals its upload limit should be marked as inactive even if it never expires",
			now:         time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: new(3),
				FilesUploaded:  3,
			},
			isActiveWant: false,
		},
		{
			explanation: "an enabled link with no expiration and an upload count below its upload limit should be marked as active",
			now:         time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: new(3),
				FilesUploaded:  2,
			},
			isActiveWant: true,
		},
		{
			explanation: "an enabled link with no expiration and an unlimited upload count should be marked as active",
			now:         time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
			},
			isActiveWant: true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			if got, want := tt.guestLink.IsActive(tt.now), tt.isActiveWant; got != want {
				t.Errorf("IsActive()=%v, want=%v", got, want)
			}
		})
	}
}
