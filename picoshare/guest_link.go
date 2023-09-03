package picoshare

import (
	"time"
)

type (
	GuestLinkID             string
	GuestLinkLabel          string
	GuestUploadMaxFileBytes *uint64
	GuestUploadCountLimit   *int

	GuestLink struct {
		ID             GuestLinkID
		Label          GuestLinkLabel
		Created        time.Time
		Expires        ExpirationTime
		MaxFileBytes   GuestUploadMaxFileBytes
		MaxFileUploads GuestUploadCountLimit
		FilesUploaded  int
	}
)

var (
	GuestUploadUnlimitedFileSize    = GuestUploadMaxFileBytes(nil)
	GuestUploadUnlimitedFileUploads = GuestUploadCountLimit(nil)
)

func (glid GuestLinkID) Empty() bool {
	return glid.String() == ""
}

func (glid GuestLinkID) String() string {
	return string(glid)
}

func (gl GuestLink) Empty() bool {
	return gl.ID.Empty()
}

func (gl GuestLink) CanAcceptMoreFiles() bool {
	if gl.MaxFileUploads == GuestUploadUnlimitedFileUploads {
		return true
	}
	return gl.FilesUploaded < *gl.MaxFileUploads
}

func (gl GuestLink) IsExpired() bool {
	if gl.Expires == NeverExpire {
		return false
	}
	return time.Now().After(time.Time(gl.Expires))
}

func (gl GuestLink) IsActive() bool {
	return !gl.IsExpired() && gl.CanAcceptMoreFiles()
}

func (label GuestLinkLabel) Empty() bool {
	return label.String() == ""
}

func (label GuestLinkLabel) String() string {
	return string(label)
}
