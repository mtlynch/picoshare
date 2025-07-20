package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mtlynch/picoshare/v2/handlers/parse"
	"github.com/mtlynch/picoshare/v2/picoshare"
	"github.com/mtlynch/picoshare/v2/random"
	"github.com/mtlynch/picoshare/v2/store"
)

const EntryIDLength = 10

// Omit visually similar characters (I,l,1), (0,O)
var entryIDCharacters = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

type (
	EntryPostResponse struct {
		ID string `json:"id"`
	}

	dbError struct {
		Err error
	}
)

func (dbe dbError) Error() string {
	return fmt.Sprintf("database error: %s", dbe.Err)
}

func (dbe dbError) Unwrap() error {
	return dbe.Err
}

func (s Server) entryPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expiration, err := s.parseExpirationFromRequest(r)
		if err != nil {
			log.Printf("invalid expiration URL parameter: %v", err)
			http.Error(w, fmt.Sprintf("Invalid expiration URL parameter: %v", err), http.StatusBadRequest)
			return
		}

		// We're intentionally not limiting the size of the request because we
		// assume that the uploading user is trusted, so they can upload files of
		// any size they want.
		id, err := s.insertFileFromRequest(r, expiration, picoshare.GuestLinkID(""))
		if err != nil {
			var de *dbError
			if errors.As(err, &de) {
				log.Printf("failed to insert uploaded file into data store: %v", err)
				http.Error(w, "failed to insert file into database", http.StatusInternalServerError)
			} else {
				log.Printf("invalid upload: %v", err)
				http.Error(w, fmt.Sprintf("invalid request: %s", err), http.StatusBadRequest)
			}
			return
		}

		respondJSON(w, EntryPostResponse{ID: id.String()})
	}
}

func (s Server) entryPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseEntryID(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing ID: %v", err)
			http.Error(w, fmt.Sprintf("bad entry ID: %v", err), http.StatusBadRequest)
			return
		}

		metadata, err := s.entryMetadataFromRequest(r)

		if err != nil {
			log.Printf("error parsing entry edit request: %v", err)
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
			return
		}

		if _, ok := err.(store.GuestLinkNotFoundError); ok {
			http.Error(w, "Invalid guest link ID", http.StatusNotFound)
			return
		}

		if err := s.getDB(r).UpdateEntryMetadata(id, metadata); err != nil {
			if _, ok := err.(store.EntryNotFoundError); ok {
				http.Error(w, "Invalid entry ID", http.StatusNotFound)
				return
			}
			log.Printf("error saving entry metadata: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save new entry data: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) guestEntryPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		guestLinkID, err := parseGuestLinkID(mux.Vars(r)["guestLinkID"])
		if err != nil {
			log.Printf("error parsing guest link ID: %v", err)
			http.Error(w, fmt.Sprintf("Invalid guest link ID: %v", err), http.StatusBadRequest)
			return
		}

		gl, err := s.getDB(r).GetGuestLink(guestLinkID)
		if _, ok := err.(store.GuestLinkNotFoundError); ok {
			http.Error(w, "Invalid guest link ID", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving guest link with ID %v: %v", guestLinkID, err)
			http.Error(w, "Failed to retrieve guest link", http.StatusInternalServerError)
			return
		}

		if !gl.IsActive() {
			http.Error(w, "Guest link is no longer active", http.StatusUnauthorized)
		}

		if gl.MaxFileBytes != picoshare.GuestUploadUnlimitedFileSize {
			// We technically allow slightly less than the user specified because
			// other fields in the request take up some space, but it's a difference
			// of only a few hundred bytes.
			r.Body = http.MaxBytesReader(w, r.Body, int64(*gl.MaxFileBytes))
		}

		id, err := s.insertFileFromRequest(r, gl.FileLifetime.ExpirationFromTime(s.clock.Now()), guestLinkID)
		if err != nil {
			var de *dbError
			if errors.As(err, &de) {
				log.Printf("failed to insert uploaded file into data store: %v", err)
				http.Error(w, "failed to insert file into database", http.StatusInternalServerError)
			} else {
				log.Printf("invalid upload: %v", err)
				http.Error(w, fmt.Sprintf("invalid request: %s", err), http.StatusBadRequest)
			}
			return
		}

		if clientAcceptsJson(r) {
			respondJSON(w, EntryPostResponse{ID: id.String()})
		} else {
			w.Header().Set("Content-Type", "text/plain")
			if _, err := fmt.Fprintf(w, "%s/-%s\r\n", baseURLFromRequest(r), id.String()); err != nil {
				log.Fatalf("failed to write HTTP response: %v", err)
			}
		}
	}
}

func (s Server) entryMetadataFromRequest(r *http.Request) (picoshare.UploadMetadata, error) {
	var payload struct {
		Filename   string `json:"filename"`
		Expiration string `json:"expiration"`
		Note       string `json:"note"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("failed to decode JSON request: %v", err)
		return picoshare.UploadMetadata{}, err
	}

	filename, err := parse.Filename(payload.Filename)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}

	// Treat an empty expiration string as NeverExpire.
	expiration := picoshare.NeverExpire
	if payload.Expiration != "" {
		expiration, err = parse.Expiration(payload.Expiration, s.clock.Now())
		if err != nil {
			return picoshare.UploadMetadata{}, err
		}
	}

	note, err := parse.FileNote(payload.Note)
	if err != nil {
		return picoshare.UploadMetadata{}, err
	}

	return picoshare.UploadMetadata{
		Filename: filename,
		Expires:  expiration,
		Note:     note,
	}, nil
}

func generateEntryID() picoshare.EntryID {
	return picoshare.EntryID(random.String(EntryIDLength, entryIDCharacters))
}

func parseEntryID(s string) (picoshare.EntryID, error) {
	if len(s) != EntryIDLength {
		return picoshare.EntryID(""), fmt.Errorf("entry ID (%v) has invalid length: got %d, want %d", s, len(s), EntryIDLength)
	}

	// We could do this outside the function and store the result.
	idCharsHash := map[rune]bool{}
	for _, c := range entryIDCharacters {
		idCharsHash[c] = true
	}

	for _, c := range s {
		if _, ok := idCharsHash[c]; !ok {
			return picoshare.EntryID(""), fmt.Errorf("entry ID (%s) contains invalid character: %v", s, c)
		}
	}
	return picoshare.EntryID(s), nil
}

func (s Server) insertFileFromRequest(r *http.Request, expiration picoshare.ExpirationTime, guestLinkID picoshare.GuestLinkID) (picoshare.EntryID, error) {
	// ParseMultipartForm can go above the limit we set, so set a conservative RAM
	// limit to avoid exhausting RAM on servers with limited resources.
	multipartMaxMemory := mibToBytes(1)
	if err := r.ParseMultipartForm(multipartMaxMemory); err != nil {
		return picoshare.EntryID(""), err
	}
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			log.Printf("failed to free multipart form resources: %v", err)
		}
	}()

	reader, metadata, err := r.FormFile("file")
	if err != nil {
		return picoshare.EntryID(""), err
	}

	fileSize, err := picoshare.FileSizeFromInt64(metadata.Size)
	if err != nil {
		return picoshare.EntryID(""), err
	}

	filename, err := parse.Filename(metadata.Filename)
	if err != nil {
		return picoshare.EntryID(""), err
	}

	contentType, err := parseContentType(metadata.Header.Get("Content-Type"))
	if err != nil {
		return picoshare.EntryID(""), err
	}

	note, err := parse.FileNote(r.FormValue("note"))
	if err != nil {
		return picoshare.EntryID(""), err
	}

	if guestLinkID != "" && note.Value != nil {
		return picoshare.EntryID(""), errors.New("guest uploads cannot have file notes")
	}

	id := generateEntryID()
	err = s.getDB(r).InsertEntry(reader,
		picoshare.UploadMetadata{
			ID:          id,
			Filename:    filename,
			ContentType: contentType,
			Note:        note,
			GuestLink: picoshare.GuestLink{
				ID: guestLinkID,
			},
			Uploaded: s.clock.Now(),
			Expires:  expiration,
			Size:     fileSize,
		})
	if err != nil {
		log.Printf("failed to save entry: %v", err)
		return picoshare.EntryID(""), dbError{err}
	}

	return id, nil
}

func parseContentType(s string) (picoshare.ContentType, error) {
	// The content type header is fairly open-ended, so we're liberal in what
	// values we accept.
	return picoshare.ContentType(s), nil
}

func (s Server) parseExpirationFromRequest(r *http.Request) (picoshare.ExpirationTime, error) {
	return parse.Expiration(r.URL.Query().Get("expiration"), s.clock.Now())
}

// mibToBytes converts an amount in MiB to an amount in bytes.
func mibToBytes(i int64) int64 {
	return i << 20
}

func clientAcceptsJson(r *http.Request) bool {
	accepts := r.Header.Get("Accept")
	return accepts == "*/*" || accepts == "application/json"
}

func baseURLFromRequest(r *http.Request) string {
	var scheme string
	// If we're running behind a proxy, assume that it's a TLS proxy.
	if r.TLS != nil || os.Getenv("PS_BEHIND_PROXY") != "" {
		scheme = "https"
	} else {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
