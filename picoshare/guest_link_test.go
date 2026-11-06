package picoshare_test

import (
	"testing"
	"time"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestGuestLinkIsActive(t *testing.T) {
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	expiry := picoshare.ExpirationTime(now.Add(24 * time.Hour))
	countLimit := 3

	for _, tt := range []struct {
		explanation string
		guestLink   picoshare.GuestLink
		now         time.Time
		want        bool
	}{
		{
			explanation: "link before expiry is active",
			guestLink:   picoshare.GuestLink{UrlExpires: expiry},
			now:         now,
			want:        true,
		},
		{
			explanation: "link at expiry is inactive",
			guestLink:   picoshare.GuestLink{UrlExpires: expiry},
			now:         now.Add(24 * time.Hour),
			want:        false,
		},
		{
			explanation: "link after expiry is inactive",
			guestLink:   picoshare.GuestLink{UrlExpires: expiry},
			now:         now.Add(48 * time.Hour),
			want:        false,
		},
		{
			explanation: "NeverExpire link is active",
			guestLink:   picoshare.GuestLink{UrlExpires: picoshare.NeverExpire},
			now:         now,
			want:        true,
		},
		{
			explanation: "disabled link is inactive",
			guestLink:   picoshare.GuestLink{UrlExpires: picoshare.NeverExpire, IsDisabled: true},
			now:         now,
			want:        false,
		},
		{
			explanation: "link at exact upload count limit is inactive",
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: &countLimit,
				FilesUploaded:  countLimit,
			},
			now:  now,
			want: false,
		},
		{
			explanation: "link with remaining upload count is active",
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: &countLimit,
				FilesUploaded:  countLimit - 1,
			},
			now:  now,
			want: true,
		},
		{
			explanation: "link with unlimited uploads is active",
			guestLink: picoshare.GuestLink{
				UrlExpires:     picoshare.NeverExpire,
				MaxFileUploads: picoshare.GuestUploadUnlimitedFileUploads,
			},
			now:  now,
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
