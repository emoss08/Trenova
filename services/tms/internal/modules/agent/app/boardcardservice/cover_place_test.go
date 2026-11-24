package boardcardservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/parties/domain/partiesboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var carrierKind = recordkind.Kind(permission.ResourceCarrier)

type placeMeasure struct {
	reading boardreading.Reading
	asked   []*boardreaderports.RecordRequest
}

func (*placeMeasure) Kind() recordkind.Kind { return carrierKind }

func (*placeMeasure) Measure() string { return partiesboard.MeasureCoverLeft }

func (f *placeMeasure) Readings(
	_ context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	f.asked = append(f.asked, req)
	out := map[pulid.ID]boardreading.Reading{}
	for _, id := range req.IDs {
		reading := f.reading
		fact := *reading.Fact
		reading.Fact = &fact
		out[id] = reading
	}

	return out, nil
}

type coverFixture struct {
	*fixture
	carrier pulid.ID
	cover   *placeMeasure
	lapsing *fakeMetric
}

func newCoverFixture() *coverFixture {
	f := &coverFixture{fixture: newFixture(), carrier: pulid.MustNew("car_")}
	days := boardcatalog.Days(6)
	f.cover = &placeMeasure{reading: boardreading.Reading{
		Value:    days,
		Standing: boardreading.Words{Text: "{0} left", Value: &days},
		Fact: &boardreading.Words{
			Text: "Auto liability ends {0}", At: 1_793_500_000, Day: true,
			Place: "with 4 of our loads now",
		},
		Window: &boardreading.WindowFacts{Left: 6 * 86400, Span: 30 * 86400},
	}}
	f.lapsing = &fakeMetric{
		name:    operationsboard.MetricCoverAgainstLoads,
		figures: map[string]*boardcatalog.Value{},
	}
	f.svc.readers.Records[carrierKind] = &fakeRecords{
		kind:  carrierKind,
		known: map[pulid.ID]boardreading.Summary{f.carrier: {Label: "Ridgeline Freight"}},
	}
	f.svc.readers.Measures[boardreaderports.MeasureKey{
		Kind: carrierKind, Measure: partiesboard.MeasureCoverLeft,
	}] = f.cover
	f.svc.readers.Metrics[operationsboard.MetricCoverAgainstLoads] = f.lapsing
	f.permissions.allowed[permission.ResourceCarrier] = true

	return f
}

func (f *coverFixture) carriersBoard(t *testing.T) *board.Board {
	t.Helper()

	return parsed(t, map[string]any{
		"title": "Cover", "verdict": "Their [cover][1].",
		"blocks": []any{map[string]any{
			"type": "records", "records": []any{f.carrier.String()}, "measure": "coverLeft",
		}},
	})
}

func lapsingBoard(t *testing.T) *board.Board {
	t.Helper()

	return parsed(t, map[string]any{
		"title": "Cover", "verdict": "Their [loads][1].",
		"blocks": []any{map[string]any{
			"type": "metric", "metric": operationsboard.MetricCoverAgainstLoads,
		}},
	})
}

func (f *coverFixture) asAgent(
	shown *board.Board,
	dataAccess permission.FieldSensitivity,
) *assistantports.BoardFillRequest {
	return &assistantports.BoardFillRequest{
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeAgent,
			PrincipalID:    pulid.MustNew("agd_"),
			OrganizationID: f.actor.OrganizationID,
			BusinessUnitID: f.actor.BusinessUnitID,
		},
		ThreadID: f.thread,
		Board:    shown,
		Agent:    &assistantports.BoardAgentRead{DataAccess: dataAccess},
	}
}

func coverFact(t *testing.T, filled *board.Filled) *boardreading.Words {
	t.Helper()
	require.Len(t, filled.Blocks[0].Rows, 1)
	measure := filled.Blocks[0].Rows[0].Measure
	require.NotNil(t, measure)
	require.NotNil(t, measure.Fact)

	return measure.Fact
}

func TestCoverLeftShowsTheLoadsOnlyToAReaderOfShipments(t *testing.T) {
	t.Parallel()

	f := newCoverFixture()

	filled, err := f.svc.Fill(t.Context(), f.request(f.carriersBoard(t)))

	require.NoError(t, err)
	assert.True(t, f.cover.asked[0].Place, "the reader is told it may count the loads")
	fact := coverFact(t, filled)
	assert.Equal(t, "Auto liability ends {0}", fact.Text)
	assert.Equal(t, "with 4 of our loads now", fact.Place)
}

func TestCoverLeftDrawsTheCoverWithoutTheLoadsForAReaderWhoMayNotReadShipments(
	t *testing.T,
) {
	t.Parallel()

	f := newCoverFixture()
	f.permissions.allowed[permission.ResourceShipment] = false

	filled, err := f.svc.Fill(t.Context(), f.request(f.carriersBoard(t)))

	require.NoError(t, err)
	assert.False(t, f.cover.asked[0].Place, "the loads are not counted at all")
	fact := coverFact(t, filled)
	assert.Equal(t, "Auto liability ends {0}", fact.Text, "the cover fact stays")
	assert.Empty(t, fact.Place,
		"a place a reader wrote anyway is cleared: no count, never a zero")
}

func TestTheLoadsAgainstCoverAreRefusedWithoutShipmentRead(t *testing.T) {
	t.Parallel()

	f := newCoverFixture()
	f.permissions.allowed[permission.ResourceShipment] = false

	err := f.svc.Check(t.Context(), f.request(lapsingBoard(t)))

	var mistake *agentruntimeports.PublishMistake
	require.ErrorAs(t, err, &mistake)

	filled, err := f.svc.Fill(t.Context(), f.request(lapsingBoard(t)))
	require.NoError(t, err)
	assert.Equal(t, 0, f.lapsing.calls)
	assert.Equal(t, board.StateUnreadable, filled.Blocks[0].State)
}

func TestTheLoadsAgainstCoverAreReadWithCarrierAndShipmentRead(t *testing.T) {
	t.Parallel()

	f := newCoverFixture()

	require.NoError(t, f.svc.Check(t.Context(), f.request(lapsingBoard(t))))
	_, err := f.svc.Fill(t.Context(), f.request(lapsingBoard(t)))
	require.NoError(t, err)
	assert.Equal(t, 1, f.lapsing.calls)
}

func TestAnAgentIsShownTheLoadsOnlyWhenItsDataAccessReachesTheirFields(t *testing.T) {
	t.Parallel()

	f := newCoverFixture()
	request := f.asAgent(f.carriersBoard(t), permission.SensitivityInternal)

	filled, err := f.svc.Fill(t.Context(), request)
	require.NoError(t, err)
	assert.Equal(t, "with 4 of our loads now", coverFact(t, filled).Place,
		"an agent's data access is never below Internal, the rank of a shipment's status")

	scope := &readScope{actor: request.Actor, reach: f.svc.reachOf(t.Context(), request)}
	above := &boardcatalog.PlacePart{
		Requires: []permission.Resource{permission.ResourceShipment},
		Reads: boardcatalog.Read(
			permission.ResourceShipmentMove, "externalDriverPhone",
		),
	}
	allowed, err := f.svc.placeAllowed(t.Context(), scope, above)
	require.NoError(t, err)
	assert.False(t, allowed, "a place reading a field ranked above the agent's data access")

	allowed, err = f.svc.placeAllowed(t.Context(), scope, &boardcatalog.PlacePart{
		Requires: []permission.Resource{permission.ResourceShipment},
		Reads:    partiesboard.LoadsReads,
	})
	require.NoError(t, err)
	assert.True(t, allowed)
}
