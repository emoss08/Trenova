package imageutils

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToDataURI(t *testing.T) {
	body := pngBytes(t, 4, 4)

	uri := ToDataURI(&Image{Data: body, Format: FormatPNG, Width: 4, Height: 4})

	assert.True(t, strings.HasPrefix(uri, "data:image/png;base64,"))
	encoded := strings.TrimPrefix(uri, "data:image/png;base64,")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.Equal(t, body, decoded)
}

func TestToDataURIEmpty(t *testing.T) {
	assert.Empty(t, ToDataURI(nil))
	assert.Empty(t, ToDataURI(&Image{Format: FormatPNG}))
}

func TestIsDataURI(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"data:image/png;base64,AAAA", true},
		{"DATA:image/png;base64,AAAA", true},
		{"  data:image/png;base64,AAAA", true},
		{"http://example.com/a.png", false},
		{"file:///etc/passwd", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDataURI(tt.in))
		})
	}
}

func TestDecodeDataURIRoundTrip(t *testing.T) {
	body := pngBytes(t, 20, 10)
	uri := ToDataURI(&Image{Data: body, Format: FormatPNG})

	img, err := DecodeDataURI(uri, 1<<20)
	require.NoError(t, err)
	assert.Equal(t, FormatPNG, img.Format)
	assert.Equal(t, 20, img.Width)
	assert.Equal(t, 10, img.Height)
}

// TestDecodeDataURIRejectsMismatchedPayload is the assertion that matters: a
// data URI claiming to be an image while carrying markup is how an inert asset
// slot becomes a script delivery channel.
func TestDecodeDataURIRejectsMismatchedPayload(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString(
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
	)

	_, err := DecodeDataURI("data:image/png;base64,"+payload, 1<<20)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestDecodeDataURIRejects(t *testing.T) {
	pngPayload := base64.StdEncoding.EncodeToString(pngBytes(t, 4, 4))

	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{"not a data uri", "http://example.com/a.png", "not a data URI"},
		{"no separator", "data:image/png;base64" + pngPayload, "payload separator"},
		{"not base64 encoded", "data:image/png,%89PNG", "base64"},
		{"svg media type", "data:image/svg+xml;base64," + pngPayload, "unsupported"},
		{"html media type", "data:text/html;base64," + pngPayload, "unsupported"},
		{"javascript media type", "data:text/javascript;base64," + pngPayload, "unsupported"},
		{"corrupt base64", "data:image/png;base64,!!!!not-base64!!!!", "decode data URI payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeDataURI(tt.in, 1<<20)
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.wantErr))
		})
	}
}

func TestDecodeDataURIEnforcesSizeCap(t *testing.T) {
	body := pngBytes(t, 400, 400)
	uri := ToDataURI(&Image{Data: body, Format: FormatPNG})
	require.Greater(t, len(body), 64)

	_, err := DecodeDataURI(uri, 64)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the 64 byte limit")
}

func TestDecodeDataURIRejectsOversizeBeforeDecoding(t *testing.T) {
	// A payload far past the cap must be refused on its encoded length, so a
	// hostile template cannot make us allocate megabytes to reject it.
	huge := "data:image/png;base64," + strings.Repeat("A", 4<<20)

	_, err := DecodeDataURI(huge, 1024)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the 1024 byte limit")
}

func TestDecodeDataURIAllowsUncappedWhenZero(t *testing.T) {
	uri := ToDataURI(&Image{Data: pngBytes(t, 8, 8), Format: FormatPNG})

	_, err := DecodeDataURI(uri, 0)
	assert.NoError(t, err)
}
