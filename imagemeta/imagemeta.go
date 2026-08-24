// Package imagemeta removes metadata such as EXIF tags from uploaded images to
// protect the privacy of the uploader.
package imagemeta

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
)

// pngSignature is the eight-byte sequence at the start of every PNG file.
var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}

// pngChunkOverhead is the number of bytes in a PNG chunk that surround its
// payload: a four-byte length, a four-byte type, and a four-byte CRC.
const pngChunkOverhead = 12

// Strip returns a reader over the contents of r with image metadata removed. It
// understands JPEG and PNG, the formats most likely to carry EXIF or other
// identifying metadata. Anything else, including input that is too short or too
// malformed to parse, is passed through unchanged so that stripping never
// corrupts or rejects an upload.
func Strip(r io.Reader) (io.Reader, error) {
	br := bufio.NewReader(r)

	// Peek never consumes, so the format bytes remain available to whichever
	// reader we return. A short read here just means the input is smaller than
	// an image header, in which case we fall through to the passthrough case. A
	// genuine read error surfaces again downstream.
	header, _ := br.Peek(len(pngSignature))

	switch {
	case isJPEG(header):
		return stripAll(br, stripJPEG)
	case isPNG(header):
		return stripAll(br, stripPNG)
	default:
		return br, nil
	}
}

func stripAll(r io.Reader, strip func([]byte) []byte) (io.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(strip(data)), nil
}

func isJPEG(header []byte) bool {
	return len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF
}

func isPNG(header []byte) bool {
	return bytes.HasPrefix(header, pngSignature)
}

// stripJPEG returns data with its application (APPn) and comment (COM) segments
// removed. Those are the segments that hold EXIF, XMP, ICC, and similar
// metadata. Every other segment, including the start-of-scan marker and the
// compressed image data that follows it, is copied byte for byte. If the data
// does not parse as a JPEG we return it untouched.
func stripJPEG(data []byte) []byte {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return data
	}

	out := make([]byte, 0, len(data))
	out = append(out, data[0], data[1]) // Start-of-image marker.

	i := 2
	for {
		if i+1 >= len(data) || data[i] != 0xFF {
			return data // Malformed, so leave the original untouched.
		}

		marker := data[i+1]

		// Extra 0xFF bytes are legal padding before a marker code.
		if marker == 0xFF {
			out = append(out, data[i])
			i++
			continue
		}

		// Standalone markers have no length or payload.
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			out = append(out, data[i], data[i+1])
			i += 2
			continue
		}

		// The start-of-scan marker is followed by the compressed image data,
		// which runs to the end of the file, so copy the remainder verbatim.
		if marker == 0xDA {
			out = append(out, data[i:]...)
			return out
		}

		// Every other marker carries a two-byte length that counts itself.
		if i+3 >= len(data) {
			return data
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if segLen < 2 || i+2+segLen > len(data) {
			return data
		}
		next := i + 2 + segLen

		if !isJPEGMetadataMarker(marker) {
			out = append(out, data[i:next]...)
		}
		i = next
	}
}

func isJPEGMetadataMarker(marker byte) bool {
	return (marker >= 0xE0 && marker <= 0xEF) || marker == 0xFE
}

// stripPNG returns data with its metadata chunks removed while preserving the
// chunks that describe the image itself. If the data does not parse as a PNG we
// return it untouched.
func stripPNG(data []byte) []byte {
	if !bytes.HasPrefix(data, pngSignature) {
		return data
	}

	out := make([]byte, 0, len(data))
	out = append(out, pngSignature...)

	i := len(pngSignature)
	for i+pngChunkOverhead <= len(data) {
		length := binary.BigEndian.Uint32(data[i : i+4])
		chunkEnd := uint64(i) + uint64(pngChunkOverhead) + uint64(length)
		if chunkEnd > uint64(len(data)) {
			return data // Truncated, so leave the original untouched.
		}

		chunkType := string(data[i+4 : i+8])
		if !isPNGMetadataChunk(chunkType) {
			out = append(out, data[i:chunkEnd]...)
		}

		i = int(chunkEnd)
		if chunkType == "IEND" {
			return out
		}
	}

	return data // No end chunk found, so leave the original untouched.
}

func isPNGMetadataChunk(chunkType string) bool {
	switch chunkType {
	case "tEXt", "zTXt", "iTXt", "tIME", "eXIf":
		return true
	default:
		return false
	}
}
