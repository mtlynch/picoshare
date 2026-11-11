package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/handlers/parse"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

type guestUploadRequest struct {
	GuestLinkID picoshare.GuestLinkID
}

func parseGuestUploadRequest(r *http.Request) (guestUploadRequest, error) {
	id, err := parseGuestLinkID(mux.Vars(r)["guestLinkID"])
	if err != nil {
		return guestUploadRequest{}, err
	}
	if r.URL.Query().Get("note") != "" || r.URL.Query().Get("downloadPassphrase") != "" {
		return guestUploadRequest{}, errors.New("guest uploads cannot have notes or download passphrases")
	}
	return guestUploadRequest{GuestLinkID: id}, nil
}

func (s Server) receiveGuestUpload(r *http.Request, reservation picoshare.GuestUploadReservation) (err error) {
	// Cleanup runs after receiving stops, without the canceled request context.
	// Never release capacity while staged chunks still occupy it.
	defer func() {
		if err != nil {
			if cleanupErr := s.store.AbortGuestUpload(reservation); cleanupErr != nil {
				err = &dbError{errors.Join(err, fmt.Errorf("failed to abort guest upload: %w", cleanupErr))}
			}
		}
	}()

	expiration, err := s.parseGuestExpirationFromRequest(r, reservation.GuestLink())
	if err != nil {
		return err
	}
	reader, err := r.MultipartReader()
	if err != nil {
		return err
	}
	metadata := picoshare.UploadMetadata{
		Uploaded: s.now(), Expires: expiration,
	}
	foundFile := false
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if part.FileName() != "" {
			if part.FormName() != "file" || foundFile {
				return errors.New("guest uploads require exactly one file part named file")
			}
			metadata.Filename, err = parse.Filename(part.FileName())
			if err != nil {
				return err
			}
			metadata.ContentType, err = parseContentType(part.Header.Get("Content-Type"))
			if err != nil {
				return err
			}
			if err := s.store.WriteGuestUpload(r.Context(), reservation, part); err != nil {
				if _, ok := errors.AsType[store.GuestUploadInvalidError](err); ok {
					return err
				}
				return &dbError{err}
			}
			foundFile = true
		} else {
			switch part.FormName() {
			case "note", "downloadPassphrase":
				// A single byte proves a forbidden field is nonempty. Don't
				// accumulate untrusted scalar fields in memory.
				n, err := io.CopyN(io.Discard, part, 1)
				if n != 0 {
					return fmt.Errorf("guest uploads cannot have %s", part.FormName())
				}
				if err != io.EOF {
					return err
				}
			default:
				if _, err := io.Copy(io.Discard, part); err != nil {
					return err
				}
			}
		}
		if err := part.Close(); err != nil {
			return err
		}
	}
	if !foundFile {
		return errors.New("missing file")
	}
	if err := s.store.CompleteGuestUpload(r.Context(), reservation, metadata); err != nil {
		if _, ok := errors.AsType[store.GuestUploadRejectedError](err); ok {
			return err
		}
		return &dbError{err}
	}
	return nil
}
