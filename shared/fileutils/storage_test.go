package fileutils

import (
	"strings"
	"testing"
	"unicode/utf8"

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

func TestSanitizeDisplayFilenameNeverCutsACharacterInHalf(t *testing.T) {
	name := SanitizeDisplayFilename("Relevé de compte", "report", 6)

	require.Equal(t, "Relev", name)
	require.True(t, utf8.ValidString(name))
}

func TestSanitizeDisplayFilenameFallsBackWhenTheCutLeavesNothing(t *testing.T) {
	require.Equal(t, "report", SanitizeDisplayFilename("日本", "report", 2))
}
