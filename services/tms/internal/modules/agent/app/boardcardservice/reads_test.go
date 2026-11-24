package boardcardservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFillReadsEachKindMeasureAndMetricOnce(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.allowed[permission.ResourceInvoice] = true
	shipments := []pulid.ID{pulid.MustNew("shp_"), pulid.MustNew("shp_"), pulid.MustNew("shp_")}
	invoice := pulid.MustNew("inv_")
	f.invoices.known[invoice] = boardreading.Summary{Label: "INV-1"}
	f.late.readings = map[pulid.ID]boardreading.Reading{}
	for idx, id := range shipments {
		f.shipments.known[id] = boardreading.Summary{Label: "S-" + id.String()[4:8]}
		f.late.readings[id] = lateness(int64(idx+1) * 600)
	}
	records := []any{shipments[0].String(), shipments[1].String()}
	others := []any{shipments[1].String(), shipments[2].String()}
	shown := parsed(t, map[string]any{
		"title": "Late and billed",
		"readings": []any{
			map[string]any{
				"key": "furthest", "label": "Furthest", "records": records,
				"measure": "lateness", "aggregate": "max",
			},
			map[string]any{"key": "active", "label": "Active", "metric": "activeShipments"},
		},
		"verdict": "The furthest is {furthest} behind of {active}, [on the lanes][1].",
		"blocks": []any{
			map[string]any{"type": "records", "records": records, "measure": "lateness"},
			map[string]any{"type": "records", "records": others, "measure": "revenue"},
			map[string]any{"type": "figures", "figures": []any{
				map[string]any{"label": "Late", "records": others, "measure": "lateness"},
				map[string]any{"label": "On time", "metric": "onTimePercent"},
			}},
			map[string]any{"type": "records", "records": []any{invoice.String()}},
		},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	assert.Equal(t, 1, f.shipments.calls, "one read for every shipment the board names")
	assert.ElementsMatch(t, shipments, f.shipments.asked[0])
	assert.Equal(t, 1, f.invoices.calls)
	assert.Equal(t, 1, f.late.calls, "one read of lateness across the blocks and the reading")
	assert.Equal(t, 1, f.revenue.calls)
	assert.Equal(t, 1, f.dispatch.calls)
	assert.Zero(t, f.receivables.calls)

	rows := filled.Blocks[0].Rows
	require.Len(t, rows, 2)
	assert.Equal(t, shipments[1], rows[0].ID, "furthest behind first")
	require.NotNil(t, rows[0].Measure.Bar)
	assert.InDelta(t, 1.0, rows[0].Measure.Bar.Segments[0].Width, 0.0001)
	assert.InDelta(t, 0.5, rows[1].Measure.Bar.Segments[0].Width, 0.0001)
	assert.Equal(t, "20m", board.FormatValue(filled.Reading("furthest").Value))
	assert.Equal(t, "41", board.FormatValue(filled.Reading("active").Value))
}
