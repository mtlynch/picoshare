package handlers

import (
	"context"
	"sync"

	"github.com/mtlynch/picoshare/picoshare"
)

// guestUploadLocks coordinates uploads admitted by one Server, not independent
// processes writing to the same store.
type guestUploadLocks struct {
	mu    sync.Mutex
	links map[picoshare.GuestLinkID]*guestUploadLock
}

func (locks *guestUploadLocks) acquire(ctx context.Context, id picoshare.GuestLinkID) (func(), error) {
	locks.mu.Lock()
	lock := locks.links[id]
	if lock == nil {
		lock = &guestUploadLock{token: make(chan struct{}, 1)}
		locks.links[id] = lock
	}
	// Include waiters so releasing a holder cannot remove a lock still in use.
	lock.references++
	locks.mu.Unlock()

	select {
	case lock.token <- struct{}{}:
		unlock := func() {
			<-lock.token
			locks.drop(id, lock)
		}
		// If cancellation and acquisition were both ready, do not admit work.
		if err := ctx.Err(); err != nil {
			unlock()
			return nil, err
		}
		return unlock, nil
	case <-ctx.Done():
		locks.drop(id, lock)
		return nil, ctx.Err()
	}
}

func (locks *guestUploadLocks) drop(id picoshare.GuestLinkID, lock *guestUploadLock) {
	locks.mu.Lock()
	defer locks.mu.Unlock()
	lock.references--
	if lock.references == 0 {
		delete(locks.links, id)
	}
}

type guestUploadLock struct {
	token      chan struct{}
	references int
}

func (s Server) acquireGuestLink(ctx context.Context, id picoshare.GuestLinkID) (picoshare.GuestLink, func(), error) {
	gl, err := s.store.GetGuestLink(id)
	if err != nil {
		return picoshare.GuestLink{}, nil, err
	}
	if gl.MaxFileUploads == picoshare.GuestUploadUnlimitedFileUploads {
		return gl, func() {}, nil
	}

	unlock, err := s.guestUploadLocks.acquire(ctx, id)
	if err != nil {
		return picoshare.GuestLink{}, nil, err
	}
	// A preceding upload may have consumed capacity while this request waited.
	// Only the refreshed link may authorize the request.
	gl, err = s.store.GetGuestLink(id)
	if err != nil {
		unlock()
		return picoshare.GuestLink{}, nil, err
	}
	return gl, unlock, nil
}
