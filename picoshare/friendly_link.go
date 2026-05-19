package picoshare

import (
	"time"
)

type FriendlyLink struct {
	FriendlyName    FriendlyName
	EntryID         EntryID
	Label           GuestLinkLabel
	Created         time.Time
	UrlExpires      ExpirationTime
	MaxFileLifetime FileLifetime
	MaxFileBytes    GuestUploadMaxFileBytes
	MaxFileUploads  GuestUploadCountLimit
	IsDisabled      bool
}
