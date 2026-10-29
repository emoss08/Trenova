package dispatchquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/weatheralert"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDispatchBoard_ReadsMovesDriversAndUrgency(t *testing.T) {
	t.Parallel()

	board := &querytooltest.FakeBoard{Board: &dispatchconsoleservice.Board{
		WindowStart: 1_790_000_000,
		WindowEnd:   1_790_000_000 + 86_400,
		Summary: &dispatchboardports.BoardSummary{
			UncoveredMoves:   1,
			LateMoves:        1,
			AvailableDrivers: 1,
		},
		Moves: []*dispatchconsoleservice.BoardMove{
			{
				BoardMove: &dispatchboardports.BoardMove{
					MoveID: pulid.MustNew(
						"smv_",
					), ShipmentID: pulid.MustNew("shp_"), ProNumber: "S1",
					CustomerName: "Acme", OriginName: "Dallas Yard", OriginCity: "Dallas", OriginState: "TX",
					OriginWindowStart: 1_790_000_000 - 1800, DestinationCity: "Houston", DestinationState: "TX",
					DestinationWindowStart: 1_790_000_000 + 7200,
				},
				Urgency: dispatchconsoleservice.UrgencyLate, MinutesToPickup: -30,
			},
			{
				BoardMove: &dispatchboardports.BoardMove{
					MoveID: pulid.MustNew(
						"smv_",
					), ProNumber: "S2", AssignedWorkerName: "Maria Ortiz",
					AssignedWorkerID: pulid.MustNew(
						"wrk_",
					), CoverageType: "driver", OriginWindowStart: 1_790_000_000 + 3600,
				},
				Urgency: dispatchconsoleservice.UrgencyNow, IsCovered: true,
			},
		},
		Drivers: []*dispatchconsoleservice.BoardDriver{{
			BoardDriver: &dispatchboardports.BoardDriver{
				WorkerID:    pulid.MustNew("wrk_"),
				FirstName:   "Dan",
				LastName:    "Lee",
				TractorCode: "T-7",
			},
			Availability:     dispatchconsoleservice.AvailabilityOpen,
			DriveRemainingMs: 6 * 3_600_000, FormattedLocation: "Waco, TX", PositionAt: 1_790_000_000 - 600,
		}},
	}}
	tool := NewGetDispatchBoardTool(board)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"hoursAhead": 12, "uncoveredOnly": true}),
	)
	require.NoError(t, err)

	view, ok := result.(*boardView)
	require.True(t, ok)
	require.Len(t, view.Moves, 2)
	assert.Equal(t, "Late", view.Moves[0].Urgency)
	assert.Equal(t, "Dallas Yard in Dallas, TX", view.Moves[0].Origin)
	assert.Equal(t, "unassigned", view.Moves[0].Coverage)
	assert.True(t, view.Moves[1].Covered)
	assert.Equal(t, "Maria Ortiz", view.Moves[1].Driver)
	require.Len(t, view.Drivers, 1)
	assert.Equal(t, "Dan Lee", view.Drivers[0].Name)
	assert.EqualValues(t, 360, view.Drivers[0].DriveRemainingMinutes)
	assert.Equal(t, 1, view.Counts["Late"])
	assert.Contains(t, view.Note, "Late 1, Now 1")
	require.NotNil(t, view.Summary)
	assert.Equal(t, 1, view.Summary.UncoveredPastPickup)

	require.NotNil(t, board.Req)
	assert.False(t, board.Req.IncludeCovered)
	assert.EqualValues(t, 12*3600, board.Req.WindowEnd-board.Req.WindowStart)
	assert.Equal(t, permission.ResourceShipmentMove, tool.Policy().Resource)
}

func TestListWeatherAlerts_KeepsSevereAndWorseByDefault(t *testing.T) {
	t.Parallel()

	expires := int64(1_790_000_000 + 7200)
	weather := &querytooltest.FakeWeather{Features: []*weatheralertports.WeatherAlertFeature{
		{
			Properties: weatheralertports.WeatherAlertFeatureProperties{
				ID:            pulid.MustNew("wal_"),
				Event:         "Ice Storm Warning",
				Severity:      "Extreme",
				AreaDesc:      "Dallas County, TX",
				AlertCategory: weatheralert.AlertCategoryWinterWeather,
				Expires:       &expires,
			},
		},
		{
			Properties: weatheralertports.WeatherAlertFeatureProperties{
				ID:       pulid.MustNew("wal_"),
				Event:    "Wind Advisory",
				Severity: "Minor",
				AreaDesc: "Harris County, TX",
			},
		},
	}}
	tool := NewListWeatherAlertsTool(weather)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))
	require.NoError(t, err)
	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]weatherAlertRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "Ice Storm Warning", rows[0].Event)
	assert.Equal(t, "winter_weather", rows[0].Category)
	assert.NotEmpty(t, rows[0].Expires)

	result, err = tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"severity": "Minor", "query": "harris"}),
	)
	require.NoError(t, err)
	outcome, ok = result.(querytool.SearchOutcome)
	require.True(t, ok)
	rows, ok = outcome.Items.([]weatherAlertRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "Wind Advisory", rows[0].Event)
}
