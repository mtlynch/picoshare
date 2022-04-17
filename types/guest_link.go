package types

import "time"

type (
	GuestLinkID             string
	GuestLinkLabel          string
	GuestUploadMaxFileBytes *uint64
	GuestUploadCountLimit   int

	GuestLink struct {
		ID                   GuestLinkID
		Label                GuestLinkLabel
		Created              time.Time
		Expires              ExpirationTime
		MaxFileBytes         GuestUploadMaxFileBytes
		UploadCountRemaining *GuestUploadCountLimit
	}
)

func (gl GuestLink) CanAcceptMoreFiles() bool {
	if gl.UploadCountRemaining == nil {
		return true
	}
	r := int(*gl.UploadCountRemaining)
	return r > 0
}

func (gl *GuestLink) DecrementUploadCount() {
	if gl.UploadCountRemaining == nil {
		return
	}
	s := GuestUploadCountLimit(int(*gl.UploadCountRemaining) - 1)
	gl.UploadCountRemaining = &s
}
