package agentevalgate_test

import (
	"flag"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite the checked-in snapshots and floors")

const catalogGolden = "testdata/catalog.golden.json"

func newKit(t *testing.T) *agentevalgate.Kit {
	t.Helper()

	kit, err := agentevalgate.NewKit()
	require.NoError(t, err)

	return kit
}

func TestToolCatalogSnapshot(t *testing.T) {
	t.Parallel()

	encoded, err := agentevalgate.MarshalJSON(newKit(t).Snapshot())
	require.NoError(t, err)

	agentevalgate.Golden(t, catalogGolden, encoded, *update,
		"go test -tags nofitz -run TestToolCatalogSnapshot "+
			"./internal/core/services/agentevalgate/ -update")
}
