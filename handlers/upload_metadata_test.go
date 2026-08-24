package handlers_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestUploadImageMetadataStripping(t *testing.T) {
	secret := []byte("secret-gps-coordinates")
	jpegData := jpegWithExif(t, secret)

	for _, tt := range []struct {
		explanation      string
		stripMetadata    bool
		metadataExpected bool
	}{
		{
			"strips metadata from the stored image when enabled",
			true,
			false,
		},
		{
			"keeps the image untouched when disabled",
			false,
			true,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New()
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceChecker, nilGarbageCollector, handlers.NewClock(), handlers.WithImageMetadataStripping(tt.stripMetadata))

			formData, contentType := createMultipartFormBody("photo.jpg", "", bytes.NewReader(jpegData))
			req := httptest.NewRequest(http.MethodPost, "/api/entry?expiration=2050-01-01T00:00:00Z", formData)
			req.Header.Add("Content-Type", contentType)

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, http.StatusOK; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			var response handlers.EntryPostResponse
			if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}

			entryFile, err := dataStore.ReadEntryFile(picoshare.EntryID(response.ID))
			if err != nil {
				t.Fatalf("failed to read stored file: %v", err)
			}
			stored := mustReadAll(entryFile)

			if got, want := bytes.Contains(stored, secret), tt.metadataExpected; got != want {
				t.Errorf("stored file contains metadata=%v, want=%v", got, want)
			}

			if _, _, err := decodeImage(stored); err != nil {
				t.Errorf("stored file is not a valid image: %v", err)
			}
		})
	}
}

// jpegWithExif encodes a small JPEG and inserts an EXIF segment carrying the
// given payload immediately after the start-of-image marker.
func jpegWithExif(t *testing.T, payload []byte) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 16), G: uint8(y * 16), B: 200, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode JPEG: %v", err)
	}
	base := buf.Bytes()

	exif := append([]byte("Exif\x00\x00"), payload...)
	segment := []byte{0xFF, 0xE1, 0x00, 0x00}
	binary.BigEndian.PutUint16(segment[2:4], uint16(len(exif)+2))
	segment = append(segment, exif...)

	out := make([]byte, 0, len(base)+len(segment))
	out = append(out, base[:2]...)
	out = append(out, segment...)
	return append(out, base[2:]...)
}

func decodeImage(data []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(data))
}
