package picoshare_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestGuestLinkIsActive(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		guestLink   picoshare.GuestLink
		now         time.Time
		want        bool
	}{
		{
			explanation: "link before expiry is active",
			guestLink: picoshare.GuestLink{
				UrlExpires: picoshare.ExpirationTime(time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)),
			},
			now:  time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			explanation: "link at expiry is inactive",
			guestLink: picoshare.GuestLink{
				UrlExpires: picoshare.ExpirationTime(time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)),
			},
			now:  time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			explanation: "link after expiry is inactive",
			guestLink: picoshare.GuestLink{
				UrlExpires: picoshare.ExpirationTime(time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)),
			},
			now:  time.Date(2025, time.January, 3, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			explanation: "NeverExpire link is active",
			guestLink: picoshare.GuestLink{
				UrlExpires: picoshare.NeverExpire,
			},
			now:  time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			explanation: "disabled link is inactive",
			guestLink: picoshare.GuestLink{
				UrlExpires: picoshare.NeverExpire,
				IsDisabled: true,
			},
			now:  time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			explanation: "link at exact upload count limit is inactive",
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: new(3),
				FilesUploaded:  3,
			},
			now:  time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			explanation: "link with remaining upload count is active",
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: new(3),
				FilesUploaded:  2,
			},
			now:  time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			explanation: "link with unlimited uploads is active",
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
			},
			now:  time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			want: true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			if got, want := tt.guestLink.IsActive(tt.now), tt.want; got != want {
				t.Errorf("IsActive()=%v, want=%v", got, want)
			}
		})
	}
}
