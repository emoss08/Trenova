package chainsnap_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/apichain/chainsnap"
	"github.com/stretchr/testify/assert"
)

const (
	shipmentID = "shp_01JABCDEFGHJKMNPQRSTVWXYZ0"
	otherID    = "shp_01JABCDEFGHJKMNPQRSTVWXYZ1"
	customerID = "cus_01JABCDEFGHJKMNPQRSTVWXYZ2"
)

func TestNormalizer_AliasesIDsInTheOrderTheyAppear(t *testing.T) {
	t.Parallel()

	n := chainsnap.NewNormalizer(map[string]string{customerID: "fixture.customer"}, nil, nil)

	got := n.Value(nil, map[string]any{
		"a": shipmentID,
		"b": []any{otherID, shipmentID},
		"c": customerID,
		"d": "workflow-" + shipmentID,
	})

	assert.Equal(t, map[string]any{
		"a": "$shp_1",
		"b": []any{"$shp_2", "$shp_1"},
		"c": "$fixture.customer",
		"d": "workflow-$shp_1",
	}, got)
}

func TestNormalizer_ReplacesTimesOnlyWhereTheKeyNamesATime(t *testing.T) {
	t.Parallel()

	n := chainsnap.NewNormalizer(nil, nil, nil)

	got := n.Value(nil, map[string]any{
		"createdAt":  float64(1_762_000_000),
		"updated_at": float64(1_762_000_000_123),
		"dueDate":    "2026-11-05T12:00:00Z",
		"weight":     float64(1_762_000_000),
		"format":     "2026-11-05T12:00:00Z",
		"latitude":   34.05,
	})

	assert.Equal(t, map[string]any{
		"createdAt":  chainsnap.TimePlaceholder,
		"updated_at": chainsnap.TimePlaceholder,
		"dueDate":    chainsnap.TimePlaceholder,
		"weight":     float64(1_762_000_000),
		"format":     "2026-11-05T12:00:00Z",
		"latitude":   34.05,
	}, got)
}

func TestNormalizer_BlanksVolatilePathsAndRunLiterals(t *testing.T) {
	t.Parallel()

	n := chainsnap.NewNormalizer(nil, []string{"data.items.*.score"},
		map[string]string{"x7k2": "<run.unique>"})

	got := n.Value(nil, map[string]any{"data": map[string]any{"items": []any{
		map[string]any{"score": 0.93, "bol": "BOL-x7k2"},
	}}})

	assert.Equal(t, map[string]any{"data": map[string]any{"items": []any{
		map[string]any{"score": chainsnap.VolatilePlaceholder, "bol": "BOL-<run.unique>"},
	}}}, got)
}

func TestIsID(t *testing.T) {
	t.Parallel()

	assert.True(t, chainsnap.IsID(shipmentID))
	assert.False(t, chainsnap.IsID("workflow-"+shipmentID))
	assert.False(t, chainsnap.IsID("shp_short"))
}
