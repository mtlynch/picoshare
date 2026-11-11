package picoshare

// GuestUploadReservation identifies one server-generated guest upload. Storage
// must verify its ownership against the persisted reservation before using it.
type GuestUploadReservation struct {
	id        EntryID
	guestLink GuestLink
}

// NewGuestUploadReservation creates a fresh identity, never a caller-chosen ID.
func NewGuestUploadReservation(link GuestLink) GuestUploadReservation {
	return GuestUploadReservation{id: NewEntryID(), guestLink: link}
}

func (r GuestUploadReservation) ID() EntryID {
	return r.id
}

func (r GuestUploadReservation) GuestLink() GuestLink {
	return r.guestLink
}
