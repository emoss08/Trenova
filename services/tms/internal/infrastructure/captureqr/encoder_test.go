package captureqr

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var finder = []string{
	"1111111",
	"1000001",
	"1011101",
	"1011101",
	"1011101",
	"1000001",
	"1111111",
}

func TestAnEncodedSheetIsASquareCodeWithItsThreeFinderPatterns(t *testing.T) {
	t.Parallel()

	payload := capturecoversheet.CoverSheetPayload("dGhpcy1pcy1hLXJhbmRvbS10b2tlbi1mb3ItdGVzdGluZw")
	code, err := New().Encode(payload)
	require.NoError(t, err)
	require.Equal(t, code.Size, len(code.Modules))
	require.Zero(t, (code.Size-21)%4, "a QR code is 21 modules plus 4 per version")
	for _, row := range code.Modules {
		require.Len(t, row, code.Size)
		assert.Empty(t, strings.Trim(row, "01"))
	}

	for _, corner := range [][2]int{{0, 0}, {code.Size - 7, 0}, {0, code.Size - 7}} {
		for dy, want := range finder {
			x, y := corner[0], corner[1]+dy
			assert.Equal(t, want, code.Modules[y][x:x+7], "finder at %v, row %d", corner, dy)
		}
	}
}

func TestEncodeRefusesAnEmptyOrOversizedPayload(t *testing.T) {
	t.Parallel()

	_, err := New().Encode("")
	require.ErrorIs(t, err, ErrPayloadInvalid)

	_, err = New().Encode(strings.Repeat("x", maxPayloadLength+1))
	require.ErrorIs(t, err, ErrPayloadInvalid)
}
