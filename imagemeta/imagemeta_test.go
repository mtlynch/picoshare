package imagemeta_test

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"github.com/mtlynch/picoshare/imagemeta"
)

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}

const pngChunkOverhead = 12

func TestStripRemovesMetadata(t *testing.T) {
	secret := []byte("SECRET-PRIVACY-DATA")
	for _, tt := range []struct {
		explanation string
		input       []byte
	}{
		{
			"removes an EXIF segment from a JPEG",
			jpegWithSegment(t, 0xE1, append([]byte("Exif\x00\x00"), secret...)),
		},
		{
			"removes a comment segment from a JPEG",
			jpegWithSegment(t, 0xFE, secret),
		},
		{
			"removes a tEXt chunk from a PNG",
			pngWithChunk(t, "tEXt", append([]byte("Comment\x00"), secret...)),
		},
		{
			"removes an eXIf chunk from a PNG",
			pngWithChunk(t, "eXIf", secret),
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			if got, want := bytes.Contains(tt.input, secret), true; got != want {
				t.Fatalf("input contains secret=%v, want=%v", got, want)
			}

			stripped := mustStrip(t, tt.input)

			if got, want := bytes.Contains(stripped, secret), false; got != want {
				t.Errorf("stripped contains secret=%v, want=%v", got, want)
			}

			if got, want := decodedBounds(t, stripped), decodedBounds(t, tt.input); got != want {
				t.Errorf("stripped image bounds=%v, want=%v", got, want)
			}
		})
	}
}

func TestStripPassesThroughUnsupportedContent(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		input       []byte
	}{
		{"plain text", []byte("this is not an image, it is plain text")},
		{"empty input", []byte{}},
		{"fewer bytes than any image header", []byte{0x00, 0x01, 0x02}},
		{"GIF data", []byte("GIF89a followed by some bytes")},
		{"PDF data", []byte("%PDF-1.7 followed by some bytes")},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			if got, want := mustStrip(t, tt.input), tt.input; !bytes.Equal(got, want) {
				t.Errorf("stripped=%v, want=%v", got, want)
			}
		})
	}
}

func TestStripLeavesMalformedImagesUntouched(t *testing.T) {
	truncatedPNG := append([]byte{}, pngSignature...)
	truncatedPNG = append(truncatedPNG, 0x00, 0x00, 0x00, 0x20, 't', 'E', 'X', 't')

	for _, tt := range []struct {
		explanation string
		input       []byte
	}{
		{
			"JPEG header with a segment length that runs past the end",
			[]byte{0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xFF, 0x00},
		},
		{
			"PNG signature with a truncated chunk",
			truncatedPNG,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			if got, want := mustStrip(t, tt.input), tt.input; !bytes.Equal(got, want) {
				t.Errorf("stripped=%v, want=%v", got, want)
			}
		})
	}
}

func TestStripIsIdempotent(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		input       []byte
	}{
		{"JPEG", jpegWithSegment(t, 0xE1, []byte("Exif\x00\x00secret"))},
		{"PNG", pngWithChunk(t, "tEXt", []byte("Comment\x00secret"))},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			once := mustStrip(t, tt.input)
			twice := mustStrip(t, once)
			if got, want := twice, once; !bytes.Equal(got, want) {
				t.Errorf("second strip changed the output: got=%v, want=%v", got, want)
			}
		})
	}
}

func mustStrip(t *testing.T, data []byte) []byte {
	t.Helper()
	r, err := imagemeta.Strip(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Strip returned error: %v", err)
	}
	stripped, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stripped output: %v", err)
	}
	return stripped
}

func decodedBounds(t *testing.T, data []byte) image.Rectangle {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode image: %v", err)
	}
	return img.Bounds()
}

// jpegWithSegment encodes a small JPEG and inserts a segment with the given
// marker and payload immediately after the start-of-image marker.
func jpegWithSegment(t *testing.T, marker byte, payload []byte) []byte {
	t.Helper()

	segment := []byte{0xFF, marker, 0x00, 0x00}
	binary.BigEndian.PutUint16(segment[2:4], uint16(len(payload)+2))
	segment = append(segment, payload...)

	return insertAt(encodeJPEG(t), 2, segment)
}

// pngWithChunk encodes a small PNG and inserts a chunk of the given type
// immediately after the IHDR chunk, which always comes first and holds a
// 13-byte payload.
func pngWithChunk(t *testing.T, chunkType string, payload []byte) []byte {
	t.Helper()

	offset := len(pngSignature) + pngChunkOverhead + 13
	return insertAt(encodePNG(t), offset, pngChunk(chunkType, payload))
}

func pngChunk(chunkType string, payload []byte) []byte {
	body := append([]byte(chunkType), payload...)

	chunk := []byte{0x00, 0x00, 0x00, 0x00}
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(payload)))
	chunk = append(chunk, body...)

	crc := []byte{0x00, 0x00, 0x00, 0x00}
	binary.BigEndian.PutUint32(crc, crc32.ChecksumIEEE(body))
	return append(chunk, crc...)
}

func encodeJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, testImage(), nil); err != nil {
		t.Fatalf("failed to encode JPEG: %v", err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, testImage()); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}
	return buf.Bytes()
}

func testImage() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 40), G: uint8(y * 40), B: 128, A: 255})
		}
	}
	return img
}

func insertAt(data []byte, offset int, insert []byte) []byte {
	out := make([]byte, 0, len(data)+len(insert))
	out = append(out, data[:offset]...)
	out = append(out, insert...)
	return append(out, data[offset:]...)
}
