// Package imageutils sniffs, normalizes, and sizes raster images destined for
// generated documents.
//
// It handles bytes only: fetching them is the caller's problem, deliberately, so
// that a package used while rendering customer-facing documents has no network
// or storage surface of its own.
package imageutils

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
)

// Format is a supported raster format.
//
// SVG is deliberately absent. It is a script vector as much as an image, and it
// buys nothing in a print context where every asset is rasterized anyway.
type Format string

const (
	FormatPNG  = Format("png")
	FormatJPEG = Format("jpeg")
	FormatWebP = Format("webp")
	FormatGIF  = Format("gif")
)

func (f Format) String() string { return string(f) }

// MIMEType returns the media type to use in a data: URI.
func (f Format) MIMEType() string {
	switch f {
	case FormatPNG:
		return "image/png"
	case FormatJPEG:
		return "image/jpeg"
	case FormatWebP:
		return "image/webp"
	case FormatGIF:
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

// FormatFromMIME maps a media type to a supported format.
func FormatFromMIME(mime string) (Format, bool) {
	normalized := strings.ToLower(strings.TrimSpace(mime))
	if idx := strings.IndexByte(normalized, ';'); idx >= 0 {
		normalized = strings.TrimSpace(normalized[:idx])
	}

	switch normalized {
	case "image/png":
		return FormatPNG, true
	case "image/jpeg", "image/jpg":
		return FormatJPEG, true
	case "image/webp":
		return FormatWebP, true
	case "image/gif":
		return FormatGIF, true
	default:
		return "", false
	}
}

var (
	// ErrEmptyImage is returned for a zero-length body.
	ErrEmptyImage = errors.New("image is empty")

	// ErrUnsupportedFormat is returned when the bytes are not a format we accept.
	ErrUnsupportedFormat = errors.New("image format is unsupported")
)

// pngMagic and friends identify a format from its leading bytes.
var (
	pngMagic  = []byte{0x89, 'P', 'N', 'G'}
	jpegMagic = []byte{0xff, 0xd8, 0xff}
	gifMagic  = []byte("GIF8")
)

// riffHeaderLength is the number of bytes needed to recognize a WebP container.
const riffHeaderLength = 12

// DetectFormat identifies the format of an image from its leading bytes.
//
// Only the bytes get a vote. A declared content type is a claim by whoever
// served the object — a storage object can be tagged application/octet-stream,
// and a remote host can label a page of markup image/png — so trusting it would
// let a non-image through on the strength of its label alone. Every format we
// accept carries an unambiguous signature, so there is nothing the claim would
// tell us that the bytes do not.
func DetectFormat(body []byte) (Format, bool) {
	switch {
	case bytes.HasPrefix(body, pngMagic):
		return FormatPNG, true
	case bytes.HasPrefix(body, jpegMagic):
		return FormatJPEG, true
	case bytes.HasPrefix(body, gifMagic):
		return FormatGIF, true
	case len(body) >= riffHeaderLength &&
		string(body[:4]) == "RIFF" &&
		string(body[8:riffHeaderLength]) == "WEBP":
		return FormatWebP, true
	default:
		return "", false
	}
}

// ErrDeclaredTypeMismatch reports that the bytes are a different image format
// than the content type claimed. The bytes win; this is for the caller to log or
// surface as a template diagnostic.
var ErrDeclaredTypeMismatch = errors.New("image bytes do not match the declared content type")

// VerifyDeclaredType compares a declared content type against the actual bytes.
//
// It returns the detected format regardless, since the bytes are authoritative;
// the error only tells the caller the claim was wrong.
func VerifyDeclaredType(body []byte, contentType string) (Format, error) {
	detected, ok := DetectFormat(body)
	if !ok {
		return "", ErrUnsupportedFormat
	}

	declared, declaredOK := FormatFromMIME(contentType)
	if declaredOK && declared != detected {
		return detected, fmt.Errorf(
			"%w: claimed %s, is %s",
			ErrDeclaredTypeMismatch,
			declared,
			detected,
		)
	}
	return detected, nil
}

// Image is a normalized raster image with known pixel dimensions.
type Image struct {
	Data   []byte
	Format Format
	Width  int
	Height int
}

// WebPTranscoder converts WebP bytes to PNG bytes.
//
// It is injected rather than imported because the only usable Go WebP decoder
// needs CGO, and this package is shared with services that must keep building
// without a C toolchain. Callers that handle WebP pass NewWebPTranscoder from a
// package that does link it.
type WebPTranscoder func(body []byte) (pngBody []byte, err error)

// Normalize validates an image and reports its dimensions, rejecting WebP.
//
// Use NormalizeWith to accept WebP.
func Normalize(body []byte) (*Image, error) {
	return NormalizeWith(body, nil)
}

// NormalizeWith validates an image and reports its dimensions.
//
// The format comes from the bytes, never from a declared content type — see
// DetectFormat.
//
// WebP is transcoded to PNG when a transcoder is supplied, because it is the one
// accepted input that print pipelines and mail clients cannot be relied on to
// render. Every other format passes through byte-for-byte: re-encoding a logo
// would cost fidelity for nothing.
//
// Decoding is a header-only operation for PNG, JPEG, and GIF, so a hostile image
// cannot make us allocate a full framebuffer to find out we do not want it.
func NormalizeWith(body []byte, webpTo WebPTranscoder) (*Image, error) {
	if len(body) == 0 {
		return nil, ErrEmptyImage
	}

	format, ok := DetectFormat(body)
	if !ok {
		return nil, ErrUnsupportedFormat
	}

	switch format {
	case FormatPNG:
		config, err := png.DecodeConfig(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("decode png header: %w", err)
		}
		return &Image{
			Data:   body,
			Format: FormatPNG,
			Width:  config.Width,
			Height: config.Height,
		}, nil

	case FormatJPEG:
		config, err := jpeg.DecodeConfig(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("decode jpeg header: %w", err)
		}
		return &Image{
			Data:   body,
			Format: FormatJPEG,
			Width:  config.Width,
			Height: config.Height,
		}, nil

	case FormatGIF:
		config, _, err := image.DecodeConfig(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("decode gif header: %w", err)
		}
		return &Image{
			Data:   body,
			Format: FormatGIF,
			Width:  config.Width,
			Height: config.Height,
		}, nil

	case FormatWebP:
		if webpTo == nil {
			return nil, fmt.Errorf("%w: webp requires a transcoder", ErrUnsupportedFormat)
		}
		converted, err := webpTo(body)
		if err != nil {
			return nil, fmt.Errorf("transcode webp to png: %w", err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(converted))
		if err != nil {
			return nil, fmt.Errorf("decode transcoded png header: %w", err)
		}
		return &Image{
			Data:   converted,
			Format: FormatPNG,
			Width:  config.Width,
			Height: config.Height,
		}, nil

	default:
		return nil, ErrUnsupportedFormat
	}
}

// FittedSize scales a source box to fit inside a bounding box without distorting
// its aspect ratio.
//
// Callers use it to emit explicit width and height on an <img>, which is what
// stops a 4000-pixel logo from pushing the rest of a document off the page.
func FittedSize(sourceWidth, sourceHeight, maxWidth, maxHeight float64) (width, height float64) {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return maxWidth, maxHeight
	}

	scale := maxWidth / sourceWidth
	if sourceHeight*scale > maxHeight {
		scale = maxHeight / sourceHeight
	}
	return sourceWidth * scale, sourceHeight * scale
}
