package cover

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProfileKeepsOnlyExecutedBlocks(t *testing.T) {
	profile := `mode: atomic
example.com/m/svc/svc.go:10.20,14.3 3 7
example.com/m/svc/svc.go:20.2,22.9 2 0
example.com/m/repo/repo.go:5.1,6.2 1 1
`

	got, err := ParseProfile(strings.NewReader(profile))
	require.NoError(t, err)

	require.Len(t, got, 2)
	assert.Equal(t, "example.com/m/repo/repo.go", got[0].ProfileName)
	assert.Equal(t, []Block{{StartLine: 5, EndLine: 6}}, got[0].Blocks)
	assert.Equal(t, "example.com/m/svc/svc.go", got[1].ProfileName)
	assert.Equal(t, []Block{{StartLine: 10, EndLine: 14}}, got[1].Blocks,
		"the zero-count block must be dropped")
}

func TestParseProfileDropsFilesWithNoExecutedBlocks(t *testing.T) {
	profile := `mode: atomic
example.com/m/cold/cold.go:1.1,2.2 1 0
`

	got, err := ParseProfile(strings.NewReader(profile))
	require.NoError(t, err)

	assert.Empty(t, got, "instrumenting the whole workspace means most files have no hits at all")
}

func TestParseProfileMergesAdjacentBlocks(t *testing.T) {
	profile := `mode: atomic
example.com/m/a/a.go:10.1,12.2 1 1
example.com/m/a/a.go:13.1,15.2 1 4
example.com/m/a/a.go:30.1,31.2 1 2
`

	got, err := ParseProfile(strings.NewReader(profile))
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, []Block{{StartLine: 10, EndLine: 15}, {StartLine: 30, EndLine: 31}}, got[0].Blocks)
}

func TestParseProfileMergesOverlappingBlocks(t *testing.T) {
	profile := `mode: atomic
example.com/m/a/a.go:10.1,20.2 1 1
example.com/m/a/a.go:12.1,15.2 1 1
`

	got, err := ParseProfile(strings.NewReader(profile))
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, []Block{{StartLine: 10, EndLine: 20}}, got[0].Blocks)
}

func TestParseProfileRejectsGarbage(t *testing.T) {
	_, err := ParseProfile(strings.NewReader("not a coverage profile\n"))

	require.Error(t, err)
}

func TestParseProfileAcceptsModeOnlyProfile(t *testing.T) {
	got, err := ParseProfile(strings.NewReader("mode: atomic\n"))

	require.NoError(t, err)
	assert.Empty(t, got, "a skipped test produces a profile with no blocks")
}

func TestBlockContains(t *testing.T) {
	b := Block{StartLine: 10, EndLine: 12}

	assert.True(t, b.Contains(10))
	assert.True(t, b.Contains(12))
	assert.False(t, b.Contains(9))
	assert.False(t, b.Contains(13))
}

func TestSplitProfileName(t *testing.T) {
	cases := []struct {
		in       string
		wantPkg  string
		wantFile string
	}{
		{"example.com/m/svc/svc.go", "example.com/m/svc", "svc.go"},
		{"example.com/m/a.go", "example.com/m", "a.go"},
		{"main.go", "", "main.go"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			pkg, file := SplitProfileName(tc.in)
			assert.Equal(t, tc.wantPkg, pkg)
			assert.Equal(t, tc.wantFile, file)
		})
	}
}
