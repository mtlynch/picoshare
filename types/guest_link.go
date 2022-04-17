package types

import "time"

type (
	GuestLinkID             string
	GuestLinkLabel          string
	GuestUploadMaxFileBytes *uint64
	GuestUploadCountLimit   *int

	GuestLink struct {
		ID                   GuestLinkID
		Label                GuestLinkLabel
		Created              time.Time
		Expires              ExpirationTime
		MaxFileBytes         GuestUploadMaxFileBytes
		UploadCountRemaining GuestUploadCountLimit
	}
)

var (
	GuestUploadUnlimitedFileSize    = GuestUploadMaxFileBytes(nil)
	GuestUploadUnlimitedFileUploads = GuestUploadCountLimit(nil)
)

func (gl GuestLink) CanAcceptMoreFiles() bool {
	if gl.UploadCountRemaining == GuestUploadUnlimitedFileUploads {
		return true
	}
	r := int(*gl.UploadCountRemaining)
	return r > 0
}

func (gl *GuestLink) DecrementUploadCount() {
	if gl.UploadCountRemaining == GuestUploadUnlimitedFileUploads {
		return
	}
	r := int(*gl.UploadCountRemaining)
	gl.UploadCountRemaining = GuestUploadCountLimit(&r)
}
