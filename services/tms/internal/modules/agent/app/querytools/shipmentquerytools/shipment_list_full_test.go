package shipmentquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentDetailOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  map[string]any
		want    Detail
		refused bool
	}{
		{name: "defaults to the summary", params: map[string]any{}, want: DetailSummary},
		{
			name:   "summary columns keep the summary",
			params: map[string]any{"fields": []any{"proNumber", "createdAt"}},
			want:   DetailSummary,
		},
		{
			name:   "a full-only column reads the full row",
			params: map[string]any{"fields": []any{"proNumber", "weight"}},
			want:   DetailFull,
		},
		{
			name:   "full asked for",
			params: map[string]any{"detail": "full"},
			want:   DetailFull,
		},
		{
			name:    "summary with a full-only column is refused",
			params:  map[string]any{"detail": "summary", "fields": []any{"stops"}},
			refused: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := shipmentDetailOf(tt.params)
			if tt.refused {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSummaryRow_HasCreatedAt(t *testing.T) {
	t.Parallel()

	assert.Contains(t, summaryColumns, "createdAt")
	assert.NotContains(t, FullOnlyColumns(), "createdAt")
}

func TestToFullRow_DescribesTheWholeLoad(t *testing.T) {
	t.Parallel()

	weight := int64(42000)
	low, high := int16(34), int16(38)
	distance := 812.5
	workerID := pulid.MustNew("wrk_")
	item := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		ProNumber:         "S1",
		Status:            shipmentvocab.StatusAssigned,
		ExternalReference: "PO-77Q",
		Weight:            &weight,
		TemperatureMin:    &low,
		TemperatureMax:    &high,
		Moves: []*shipment.ShipmentMove{{
			Status:   shipmentvocab.MoveStatusAssigned,
			Distance: &distance,
			Assignment: &shipment.Assignment{
				PrimaryWorkerID: &workerID,
				PrimaryWorker:   &worker.Worker{FirstName: "Ana", LastName: "Ruiz"},
				SecondaryWorker: &worker.Worker{FirstName: "Bo", LastName: "Lee"},
			},
			Stops: []*shipment.Stop{
				{Type: shipmentvocab.StopTypePickup, Status: shipmentvocab.StopStatusNew},
				{Type: shipmentvocab.StopTypeDelivery, Status: shipmentvocab.StopStatusNew},
			},
		}},
		Commodities: []*shipment.ShipmentCommodity{{
			Pieces: 10,
			Commodity: &commodity.Commodity{
				Name:              "Batteries",
				HazardousMaterial: &hazardousmaterial.HazardousMaterial{Name: "Lithium ion"},
			},
		}},
	}

	full := ToFullRow(ToRow(item))

	assert.Equal(t, "S1", full.ProNumber)
	assert.Equal(t, "PO-77Q", full.CustomerReference)
	assert.Equal(t, &weight, full.Weight)
	assert.Equal(t, "34 to 38", full.Temperature)
	assert.Equal(t, "812.5", full.Miles)
	require.Len(t, full.Stops, 2)
	require.Len(t, full.Moves, 1)
	assert.Contains(t, full.Moves[0], "Ana Ruiz and Bo Lee")
	assert.Equal(t, []string{"Batteries · 10 pieces · hazmat Lithium ion"}, full.Commodities)
	assert.Nil(t, full.CanceledAt)
}

func TestWithFullRows_KeepsHoldsColumnsAndTaint(t *testing.T) {
	t.Parallel()

	edi := ToRow(&shipment.Shipment{
		ID:          pulid.MustNew("shp_"),
		EntryMethod: shipmentvocab.EntryMethodEDI,
	})
	edi.ActiveHolds = []shipmentRowHold{{HoldID: "hold"}}

	result := withFullRows(querytool.SearchOutcome{Count: 1, Items: []any{edi}})
	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]FullRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Len(t, rows[0].ActiveHolds, 1)
	assert.Contains(t, outcome.Columns, "stops")
	assert.Contains(t, outcome.Columns, "proNumber")

	marked, ok := withOutsideRows(outcome).(agent.TaintCarrier)
	require.True(t, ok)
	assert.Len(t, marked.TaintedRecords(), 1)
}
