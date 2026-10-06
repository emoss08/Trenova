package qrutils

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPNGDataURIRendersASquareImage(t *testing.T) {
	t.Parallel()

	uri, err := PNGDataURI("otpauth://totp/Trenova:ada@example.com?secret=ABC", 200)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(uri, pngDataURIPrefix))

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, pngDataURIPrefix))
	require.NoError(t, err)

	img, err := png.Decode(bytes.NewReader(raw))
	require.NoError(t, err)
	assert.Equal(t, 200, img.Bounds().Dx())
	assert.Equal(t, 200, img.Bounds().Dy())
}

func TestPNGDataURIRejectsBadInput(t *testing.T) {
	t.Parallel()

	_, err := PNGDataURI("", 200)
	require.ErrorIs(t, err, ErrEmptyContent)

	_, err = PNGDataURI("content", 0)
	require.Error(t, err)
}
