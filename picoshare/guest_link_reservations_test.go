package picoshare_test

import (
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestGuestLinkCanAcceptMoreFilesIncludesReservations(t *testing.T) {
	guestLink := picoshare.GuestLink{
		MaxFileUploads:  picoshare.GuestUploadCountLimit(new(3)),
		FilesUploaded:   2,
		UploadsReserved: 1,
	}

	if got, want := guestLink.CanAcceptMoreFiles(), false; got != want {
		t.Errorf("CanAcceptMoreFiles()=%v, want=%v", got, want)
	}
}
