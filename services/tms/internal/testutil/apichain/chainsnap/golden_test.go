package chainsnap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/apichain/chainsnap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func snapshot(total string) *chainsnap.Snapshot {
	return &chainsnap.Snapshot{
		Chain: "demo",
		Steps: []chainsnap.StepRecord{
			{
				ID: "create", Call: "graphql CreateShipment", Write: true, Status: 200,
				Response: map[string]any{"data": map[string]any{"id": "$shp_1"}},
			},
			{
				ID: "charge", Call: "POST /api/v1/charges/", Write: true, Status: 200,
				Request: map[string]any{"body": map[string]any{"shipmentId": "$shp_1"}},
			},
			{
				ID: "read", Call: "graphql Shipment", Status: 200,
				Request:  map[string]any{"variables": map[string]any{"id": "$shp_1"}},
				Response: map[string]any{"data": map[string]any{"total": total}},
			},
		},
	}
}

func TestCheck_PassesWhenTheGoldenMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.golden.json")
	raw, err := chainsnap.Encode(snapshot("10.00"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	outcome, err := chainsnap.Check(path, snapshot("10.00"))

	require.NoError(t, err)
	assert.Empty(t, outcome.Diffs)
	assert.False(t, outcome.Updated)
}

func TestCheck_NamesTheStepThePathAndTheSuspectWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.golden.json")
	raw, err := chainsnap.Encode(snapshot("10.00"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	drifted := snapshot("12.00")

	outcome, err := chainsnap.Check(path, drifted)

	require.NoError(t, err)
	require.Len(t, outcome.Diffs, 1)
	diff := outcome.Diffs[0]
	assert.Equal(t, "read", diff.Step)
	assert.Equal(t, "graphql Shipment", diff.Call)
	assert.Equal(t, `response.data.total want="10.00" got="12.00"`, diff.String())
	assert.Equal(t, []string{
		"charge (POST /api/v1/charges/)",
		"create (graphql CreateShipment)",
	}, chainsnap.Suspects(drifted, diff))
}

func TestCheck_WritesTheGoldenWhenAskedTo(t *testing.T) {
	t.Setenv(chainsnap.UpdateEnv, "1")
	path := filepath.Join(t.TempDir(), "demo.golden.json")

	outcome, err := chainsnap.Check(path, snapshot("10.00"))

	require.NoError(t, err)
	assert.True(t, outcome.Updated)
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	want, err := chainsnap.Encode(snapshot("10.00"))
	require.NoError(t, err)
	assert.Equal(t, string(want), string(written))
}

func TestCheck_SaysHowToWriteAMissingGolden(t *testing.T) {
	_, err := chainsnap.Check(filepath.Join(t.TempDir(), "none.golden.json"), snapshot("1"))

	require.ErrorContains(t, err, chainsnap.UpdateEnv+"=1")
}

func TestCheck_ReportsAnAbsentStep(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.golden.json")
	raw, err := chainsnap.Encode(snapshot("10.00"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	shorter := snapshot("10.00")
	shorter.Steps = shorter.Steps[:2]

	outcome, err := chainsnap.Check(path, shorter)

	require.NoError(t, err)
	require.NotEmpty(t, outcome.Diffs)
	assert.Contains(t, outcome.Diffs[0].String(), "got=<absent>")
}
