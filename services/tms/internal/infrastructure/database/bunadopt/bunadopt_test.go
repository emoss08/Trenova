package bunadopt

import (
	"testing"
	"testing/fstest"

	"github.com/emoss08/keel/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMigrations(t *testing.T, versions ...string) []migrate.Migration {
	t.Helper()

	fsys := fstest.MapFS{}
	for _, v := range versions {
		fsys[v+"_m"+v+".sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}
	}
	fsys["R_views.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}

	ms, err := migrate.Sources(migrate.Source{Name: "core", FS: fsys}).Migrations()
	require.NoError(t, err)

	return ms
}

func appliedSet(versions ...string) map[string]bool {
	out := make(map[string]bool, len(versions))
	for _, v := range versions {
		out[v] = true
	}

	return out
}

func TestBaselineVersionIsTheNewestVersionBunApplied(t *testing.T) {
	t.Parallel()

	ms := testMigrations(t, "20240101000000", "20240101000100", "20240101000200")

	version, err := baselineVersion(ms, appliedSet("20240101000000", "20240101000100"))
	require.NoError(t, err)
	assert.Equal(t, "20240101000100", version, "the newer migration stays pending")
}

func TestBaselineVersionRefusesAGapBunLeft(t *testing.T) {
	t.Parallel()

	ms := testMigrations(t, "20240101000000", "20240101000100", "20240101000200")

	_, err := baselineVersion(ms, appliedSet("20240101000000", "20240101000200"))
	require.ErrorIs(t, err, ErrUnadoptable)
	assert.Contains(t, err.Error(), "20240101000100_m20240101000100")
}

func TestBaselineVersionRefusesAVersionTheBinaryDoesNotKnow(t *testing.T) {
	t.Parallel()

	ms := testMigrations(t, "20240101000000")

	_, err := baselineVersion(ms, appliedSet("20240101000000", "20250101000000"))
	require.ErrorIs(t, err, ErrUnadoptable)
	assert.Contains(t, err.Error(), "20250101000000")
}

func TestListedShortensALongList(t *testing.T) {
	t.Parallel()

	names := make([]string, 0, maxNamesShown+3)
	for range maxNamesShown + 3 {
		names = append(names, "m")
	}

	assert.Contains(t, listed(names), "and 3 more")
	assert.Equal(t, "a, b", listed([]string{"a", "b"}))
}
