package capturecoversheet_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoverSheetToken(t *testing.T) {
	t.Parallel()

	token, ok := capturecoversheet.CoverSheetToken(capturecoversheet.CoverSheetPayload("abc"))
	require.True(t, ok)
	assert.Equal(t, "abc", token)

	_, ok = capturecoversheet.CoverSheetToken("https://tracking.example/abc")
	assert.False(t, ok)

	_, ok = capturecoversheet.CoverSheetToken(capturecoversheet.CoverSheetPrefix)
	assert.False(t, ok, "a prefix with no token is not a sheet")
}
