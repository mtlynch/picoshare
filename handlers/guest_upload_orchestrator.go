package handlers

import (
	"errors"
	"sync"

	"github.com/mtlynch/picoshare/picoshare"
)

var errParallelGuestUpload = errors.New("parallel uploads are forbidden on this guest link")

type guestLinkStore interface {
	GetGuestLink(picoshare.GuestLinkID) (picoshare.GuestLink, error)
}

// For guests that have upload limits, claim exclusive access to prevent
// parallel uploads from exceeding the limit. Claims are local to one Server.
type guestUploadOrchestrator struct {
	store  guestLinkStore
	mu     sync.Mutex
	active map[picoshare.GuestLinkID]struct{}
}

func newGuestUploadOrchestrator(store guestLinkStore) *guestUploadOrchestrator {
	return &guestUploadOrchestrator{
		store:  store,
		active: make(map[picoshare.GuestLinkID]struct{}),
	}
}

func (locks *guestUploadOrchestrator) tryAcquire(id picoshare.GuestLinkID) (func(), bool) {
	locks.mu.Lock()
	defer locks.mu.Unlock()
	if _, active := locks.active[id]; active {
		return nil, false
	}
	locks.active[id] = struct{}{}
	return func() {
		locks.mu.Lock()
		defer locks.mu.Unlock()
		delete(locks.active, id)
	}, true
}

func (locks *guestUploadOrchestrator) acquire(id picoshare.GuestLinkID) (picoshare.GuestLink, func(), error) {
	gl, err := locks.store.GetGuestLink(id)
	if err != nil {
		return picoshare.GuestLink{}, nil, err
	}
	if gl.MaxFileUploads == picoshare.GuestUploadUnlimitedFileUploads {
		return gl, func() {}, nil
	}

	unlock, acquired := locks.tryAcquire(id)
	if !acquired {
		return picoshare.GuestLink{}, nil, errParallelGuestUpload
	}
	// Another upload may have finished after the initial lookup but before this
	// request claimed access. Only the refreshed link may authorize the request.
	gl, err = locks.store.GetGuestLink(id)
	if err != nil {
		unlock()
		return picoshare.GuestLink{}, nil, err
	}
	return gl, unlock, nil
}
