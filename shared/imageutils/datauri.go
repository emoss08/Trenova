package imageutils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// DataURIPrefix is the scheme every inlined asset carries.
const DataURIPrefix = "data:"

// base64Overhead is the numerator/denominator pair describing base64 growth,
// used to size the encode buffer without a second allocation.
const (
	base64Numerator   = 4
	base64Denominator = 3
)

// ErrNotDataURI is returned when a value is not a data: URI at all.
var ErrNotDataURI = errors.New("not a data URI")

// ToDataURI encodes an image as a base64 data: URI.
//
// Documents reference assets only this way. Nothing remote reaches the renderer,
// so a template cannot be used to probe the internal network or to signal an
// external server that a particular invoice was just printed.
func ToDataURI(img *Image) string {
	if img == nil || len(img.Data) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(img.Data)

	var b strings.Builder
	b.Grow(len(DataURIPrefix) + len(img.Format.MIMEType()) + len(";base64,") + len(encoded))
	b.WriteString(DataURIPrefix)
	b.WriteString(img.Format.MIMEType())
	b.WriteString(";base64,")
	b.WriteString(encoded)

	return b.String()
}

// IsDataURI reports whether a URL is a data: URI, case-insensitively.
func IsDataURI(raw string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), DataURIPrefix)
}

// DecodeDataURI extracts and validates the payload of an image data: URI.
//
// It exists so a template that already carries an inline image is held to the
// same standard as one we fetch: the declared media type must be a raster format
// we accept, and the decoded bytes must actually be that image. A data: URI
// claiming image/png while carrying markup is exactly how an inert asset slot
// turns into a script delivery channel.
func DecodeDataURI(raw string, maxBytes int64) (*Image, error) {
	return DecodeDataURIWith(raw, maxBytes, nil)
}

// DecodeDataURIWith is DecodeDataURI with WebP support, given a transcoder.
func DecodeDataURIWith(raw string, maxBytes int64, webpTo WebPTranscoder) (*Image, error) {
	trimmed := strings.TrimSpace(raw)
	if !IsDataURI(trimmed) {
		return nil, ErrNotDataURI
	}

	meta, payload, found := strings.Cut(trimmed[len(DataURIPrefix):], ",")
	if !found {
		return nil, errors.New("data URI has no payload separator")
	}

	if !strings.Contains(strings.ToLower(meta), "base64") {
		// Percent-encoded data URIs are legal but nothing generates them here,
		// and accepting a second encoding is a second place to get it wrong.
		return nil, errors.New("data URI must be base64 encoded")
	}

	mime, _, _ := strings.Cut(meta, ";")
	if _, ok := FormatFromMIME(mime); !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, mime)
	}

	// Reject on the encoded length first, so an oversized payload never gets
	// decoded into memory.
	if maxBytes > 0 && int64(len(payload)) > (maxBytes*base64Numerator/base64Denominator)+4 {
		return nil, fmt.Errorf("data URI exceeds the %d byte limit", maxBytes)
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode data URI payload: %w", err)
	}
	if maxBytes > 0 && int64(len(decoded)) > maxBytes {
		return nil, fmt.Errorf("data URI exceeds the %d byte limit", maxBytes)
	}

	// The media type above was only a gate; the bytes decide the actual format.
	return NormalizeWith(decoded, webpTo)
}
