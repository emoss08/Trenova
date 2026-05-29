package fileutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentDispositionSanitizesFilename(t *testing.T) {
	t.Parallel()

	header := ContentDisposition("attachment", "../reports/quarterly\r\nX-Test: injected.pdf")

	require.True(t, strings.HasPrefix(header, "attachment;"))
	require.Contains(t, header, "quarterlyX-Test: injected.pdf")
	require.NotContains(t, header, "\r")
	require.NotContains(t, header, "\n")
	require.NotContains(t, header, "../")
}

func TestContentDispositionDefaultsFilename(t *testing.T) {
	t.Parallel()

	header := ContentDisposition("", "")

	require.Equal(t, `attachment; filename=download`, header)
}
