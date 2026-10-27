package recordsummaryrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_793_000_000)

func at(seconds int64) *int64 { return &seconds }

func TestSummaryQueries_AreScopedToTheTenant(t *testing.T) {
	t.Parallel()

	ids := []pulid.ID{pulid.MustNew("rec_"), pulid.MustNew("rec_")}
	queries := map[string]projection{
		"shipments": shipmentsQuery(ids),
		"stops":     stopsQuery(ids),
		"invoices":  invoicesQuery(ids),
		"entries":   entriesQuery(ids),
	}
	for resource, query := range plain {
		queries[string(resource)] = query(ids)
	}

	for name, query := range queries {
		sql, args := keeltest.Render(t, query)

		assert.Contains(t, args, keeltest.TestOrganizationID.String(), name)
		assert.Contains(t, args, keeltest.TestBusinessUnitID.String(), name)
		assert.Contains(t, sql, "organization_id", name)
		assert.Contains(t, sql, "business_unit_id", name)
	}
}

func TestPlainSummaries_CoverEveryRowKindWithoutMeasures(t *testing.T) {
	t.Parallel()

	for _, resource := range board.Resources() {
		if len(board.MeasuresOf(resource)) > 0 {
			continue
		}
		_, ok := plain[resource]
		assert.True(t, ok, "%s has no summary query", resource)
	}
	for resource := range plain {
		assert.True(t, board.Supports(resource), "%s is summarized but not drawn", resource)
	}
	assert.NotContains(t, plain, permission.ResourceShipment)
}

func stop(
	kind shipment.StopType,
	status shipment.StopStatus,
	end int64,
	arrival *int64,
	city string,
) stopRow {
	return stopRow{
		Type: string(kind), Status: string(status), WindowStart: end - 3600,
		WindowEnd: at(end), Arrival: arrival, City: city, State: "TX",
	}
}

func TestLateness_MeasuresTheNextStopAgainstItsWindow(t *testing.T) {
	t.Parallel()

	pickedUp := stop(shipment.StopTypePickup, shipment.StopStatusCompleted, now-20000, at(now-21000), "Dallas")

	cases := map[string]struct {
		route   []stopRow
		state   board.MeasureState
		behind  int64
		tone    board.Tone
		measure bool
	}{
		"on the road and past the window": {
			route: []stopRow{pickedUp,
				stop(shipment.StopTypeDelivery, shipment.StopStatusNew, now-6000, nil, "Memphis")},
			state: board.MeasureLate, behind: 6000, tone: board.ToneWarn, measure: true,
		},
		"waiting at the stop past the window": {
			route: []stopRow{pickedUp,
				stop(shipment.StopTypeDelivery, shipment.StopStatusInTransit, now-1800, at(now-2400), "Memphis")},
			state: board.MeasureHeld, behind: 1800, tone: board.ToneWarn, measure: true,
		},
		"before the window closes": {
			route: []stopRow{pickedUp,
				stop(shipment.StopTypeDelivery, shipment.StopStatusNew, now+3600, nil, "Memphis")},
			state: board.MeasureOnTime, behind: 0, tone: board.ToneOK, measure: true,
		},
		"delivered after the window": {
			route: []stopRow{pickedUp,
				stop(shipment.StopTypeDelivery, shipment.StopStatusCompleted, now-9000, at(now-6000), "Memphis")},
			state: board.MeasureDelivered, behind: 3000, tone: board.ToneInfo, measure: true,
		},
		"delivered in the window": {
			route: []stopRow{pickedUp,
				stop(shipment.StopTypeDelivery, shipment.StopStatusCompleted, now-6000, at(now-9000), "Memphis")},
			state: board.MeasureDelivered, behind: 0, tone: board.ToneOK, measure: true,
		},
		"no stops": {measure: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			measure, ok := lateness(tc.route, now)

			require.Equal(t, tc.measure, ok)
			if !ok {
				return
			}
			assert.Equal(t, tc.state, measure.State)
			assert.Equal(t, tc.behind, measure.Value.Number.IntPart())
			assert.Equal(t, tc.tone, measure.Tone)
			assert.Equal(t, board.ValueDuration, measure.Value.Kind)
		})
	}
}

func TestLane_RunsFromTheFirstPickupToTheLastDelivery(t *testing.T) {
	t.Parallel()

	route := []stopRow{
		stop(shipment.StopTypePickup, shipment.StopStatusCompleted, now, nil, "Dallas"),
		stop(shipment.StopTypeSplitPickup, shipment.StopStatusCompleted, now, nil, "Fort Worth"),
		stop(shipment.StopTypeDelivery, shipment.StopStatusNew, now, nil, "Shreveport"),
		{Type: string(shipment.StopTypeDelivery), City: "Memphis", State: "TN"},
	}

	assert.Equal(t, "Dallas, TX → Memphis, TN", lane(route))
	assert.Equal(t, "Dallas, TX", lane(route[:1]))
	assert.Empty(t, lane(nil))
}

func TestWaiting_HoldsEveryEntryNotYetPosted(t *testing.T) {
	t.Parallel()

	assert.True(t, waiting("Draft"))
	assert.True(t, waiting("Pending"))
	assert.True(t, waiting("Approved"))
	assert.False(t, waiting("Posted"))
	assert.False(t, waiting("Reversed"))
}

func TestMoney_ReadsMinorUnitsInTheirCurrency(t *testing.T) {
	t.Parallel()

	value := money(4821055, " usd ")

	assert.Equal(t, "48210.55", value.Number.String())
	assert.Equal(t, "USD", value.Currency)
	assert.Equal(t, board.ValueMoney, value.Kind)
}
