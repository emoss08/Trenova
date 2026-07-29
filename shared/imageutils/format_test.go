package imageutils

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func solidImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 0x22, G: 0x44, B: 0x88, A: 0xff})
		}
	}
	return img
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, solidImage(w, h)))
	return buf.Bytes()
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, solidImage(w, h), nil))
	return buf.Bytes()
}

func gifBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, gif.Encode(&buf, solidImage(w, h), nil))
	return buf.Bytes()
}

func webpContainer() []byte {
	// RIFF....WEBP is the container signature; the payload is irrelevant to
	// detection, which only reads the header.
	body := append([]byte("RIFF"), []byte{0, 0, 0, 0}...)
	return append(body, []byte("WEBPVP8 ")...)
}

func TestDetectFormatReadsBytesOnly(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		want   Format
		wantOK bool
	}{
		{"png", pngBytes(t, 4, 4), FormatPNG, true},
		{"jpeg", jpegBytes(t, 4, 4), FormatJPEG, true},
		{"gif", gifBytes(t, 4, 4), FormatGIF, true},
		{"webp", webpContainer(), FormatWebP, true},
		// None of these carry an accepted signature, so no content type could
		// make them acceptable.
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), "", false},
		{"html", []byte("<html><body>x</body></html>"), "", false},
		{"pdf", []byte("%PDF-1.7\n"), "", false},
		{"random bytes", []byte{0x01, 0x02, 0x03, 0x04}, "", false},
		{"empty", nil, "", false},
		{"truncated riff", []byte("RIFF"), "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectFormat(tt.body)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestVerifyDeclaredType covers the case that motivated dropping content-type
// trust: a host serving markup while labelling it an image.
func TestVerifyDeclaredType(t *testing.T) {
	pngBody := pngBytes(t, 4, 4)

	t.Run("matching claim", func(t *testing.T) {
		format, err := VerifyDeclaredType(pngBody, "image/png")
		require.NoError(t, err)
		assert.Equal(t, FormatPNG, format)
	})

	t.Run("absent claim is not a mismatch", func(t *testing.T) {
		format, err := VerifyDeclaredType(pngBody, "application/octet-stream")
		require.NoError(t, err)
		assert.Equal(t, FormatPNG, format)
	})

	t.Run("wrong claim reports the mismatch but trusts the bytes", func(t *testing.T) {
		format, err := VerifyDeclaredType(pngBody, "image/jpeg")
		require.ErrorIs(t, err, ErrDeclaredTypeMismatch)
		assert.Equal(t, FormatPNG, format, "the bytes stay authoritative")
		assert.Contains(t, err.Error(), "claimed jpeg, is png")
	})

	t.Run("markup labelled as an image is refused outright", func(t *testing.T) {
		_, err := VerifyDeclaredType([]byte("<html><body>x</body></html>"), "image/png")
		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})
}

func TestNormalize(t *testing.T) {
	t.Run("png passes through byte for byte", func(t *testing.T) {
		body := pngBytes(t, 120, 40)

		img, err := Normalize(body)
		require.NoError(t, err)
		assert.Equal(t, FormatPNG, img.Format)
		assert.Equal(t, 120, img.Width)
		assert.Equal(t, 40, img.Height)
		assert.Equal(t, body, img.Data, "a logo must not be re-encoded")
	})

	t.Run("jpeg reports dimensions", func(t *testing.T) {
		img, err := Normalize(jpegBytes(t, 64, 16))
		require.NoError(t, err)
		assert.Equal(t, FormatJPEG, img.Format)
		assert.Equal(t, 64, img.Width)
		assert.Equal(t, 16, img.Height)
	})

	t.Run("gif reports dimensions", func(t *testing.T) {
		img, err := Normalize(gifBytes(t, 32, 8))
		require.NoError(t, err)
		assert.Equal(t, FormatGIF, img.Format)
		assert.Equal(t, 32, img.Width)
	})

	t.Run("empty body", func(t *testing.T) {
		_, err := Normalize(nil)
		assert.ErrorIs(t, err, ErrEmptyImage)
	})

	t.Run("unsupported format", func(t *testing.T) {
		_, err := Normalize([]byte("not an image at all"))
		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("truncated png fails on the header", func(t *testing.T) {
		body := pngBytes(t, 8, 8)
		_, err := Normalize(body[:12])
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode png header")
	})

	t.Run("webp without a transcoder is refused, not silently dropped", func(t *testing.T) {
		_, err := Normalize(webpContainer())
		require.ErrorIs(t, err, ErrUnsupportedFormat)
		assert.Contains(t, err.Error(), "transcoder")
	})

	t.Run("webp with a transcoder becomes png", func(t *testing.T) {
		converted := pngBytes(t, 50, 20)

		img, err := NormalizeWith(webpContainer(), func([]byte) ([]byte, error) {
			return converted, nil
		})
		require.NoError(t, err)
		assert.Equal(t, FormatPNG, img.Format, "webp must not survive into a document")
		assert.Equal(t, 50, img.Width)
		assert.Equal(t, 20, img.Height)
	})
}

func TestFittedSize(t *testing.T) {
	tests := []struct {
		name                   string
		srcW, srcH, maxW, maxH float64
		wantW, wantH           float64
	}{
		{"wide image is width bound", 400, 100, 72, 11, 44, 11},
		{"tall image is height bound", 100, 400, 72, 11, 2.75, 11},
		{"already smaller still scales to fit the box", 36, 6, 72, 11, 66, 11},
		{"square in a square", 100, 100, 50, 50, 50, 50},
		// A zero-dimension source would otherwise divide by zero.
		{"zero width falls back to the box", 0, 100, 72, 11, 72, 11},
		{"zero height falls back to the box", 100, 0, 72, 11, 72, 11},
		{"negative falls back to the box", -5, -5, 72, 11, 72, 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotW, gotH := FittedSize(tt.srcW, tt.srcH, tt.maxW, tt.maxH)
			assert.InDelta(t, tt.wantW, gotW, 0.01)
			assert.InDelta(t, tt.wantH, gotH, 0.01)
			if tt.srcW > 0 && tt.srcH > 0 {
				assert.LessOrEqual(t, gotW, tt.maxW+0.01, "must never exceed the box width")
				assert.LessOrEqual(t, gotH, tt.maxH+0.01, "must never exceed the box height")
			}
		})
	}
}

func TestFittedSizePreservesAspectRatio(t *testing.T) {
	// A 4000px logo is the case that used to push content off the page.
	gotW, gotH := FittedSize(4000, 1000, 72, 11)
	assert.InDelta(t, 4.0, gotW/gotH, 0.01)
	assert.LessOrEqual(t, gotW, 72.0)
	assert.LessOrEqual(t, gotH, 11.0)
}
