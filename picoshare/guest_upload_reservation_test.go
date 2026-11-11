package picoshare_test

import (
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
)

func TestNewGuestUploadReservation(t *testing.T) {
	link := picoshare.GuestLink{ID: "guest", MaxFileBytes: new(uint64(5))}
	first := picoshare.NewGuestUploadReservation(link)
	second := picoshare.NewGuestUploadReservation(link)
	if _, err := picoshare.EntryIDFromString(first.ID().String()); err != nil {
		t.Fatalf("generated entry ID is invalid: %v", err)
	}
	if got, want := first.GuestLink().ID, link.ID; got != want {
		t.Errorf("guest link=%s, want=%s", got, want)
	}
	if got, want := *first.GuestLink().MaxFileBytes, uint64(5); got != want {
		t.Errorf("maximum bytes=%d, want=%d", got, want)
	}
	if first.ID() == second.ID() {
		t.Error("fresh reservations must have distinct entry IDs")
	}
}
