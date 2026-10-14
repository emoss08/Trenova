package formula_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

var updateDescription = flag.Bool(
	"update-description",
	false,
	"rewrite testdata/shipment.describe.json from DescribeSchema",
)

const describedShipmentFixture = "testdata/shipment.describe.json"

func TestDescribeSchema_TheEditorFixtureIsWhatTheServerPublishes(t *testing.T) {
	t.Parallel()

	description, err := setupService(t).DescribeSchema("shipment")
	require.NoError(t, err)

	published, err := sonic.ConfigStd.MarshalIndent(description, "", "  ")
	require.NoError(t, err)
	published = append(published, '\n')

	if *updateDescription {
		require.NoError(t, os.MkdirAll(filepath.Dir(describedShipmentFixture), 0o755))
		require.NoError(t, os.WriteFile(describedShipmentFixture, published, 0o600))
	}

	fixture, err := os.ReadFile(describedShipmentFixture)
	require.NoError(t, err, "run go test -run TheEditorFixture -update-description to write it")
	require.Equal(
		t,
		string(fixture),
		string(published),
		"the editor tests read this file as the server's schema; regenerate it with -update-description",
	)
}
