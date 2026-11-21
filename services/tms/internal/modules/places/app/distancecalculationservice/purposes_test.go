package distancecalculationservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage"
	"github.com/emoss08/trenova/internal/modules/places/ports/storedmileageports"
	"github.com/emoss08/trenova/internal/testutil/mocks/distancecalculationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentdistanceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentmoveportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (f *resolveFixture) purposeProfile(routingType string) *distanceprofile.DistanceProfile {
	profile := distanceprofile.NewDefault(f.entity.OrganizationID, f.entity.BusinessUnitID)
	profile.ID = pulid.MustNew("dp_")
	profile.Name = routingType + " miles"
	profile.RoutingType = routingType
	profile.IsDefault = false
	return profile
}

func purposeRuntime(
	client mileageClient,
	profile *distanceprofile.DistanceProfile,
	stateReport bool,
) pcmilerRuntime {
	options := profile.RouteOptions()
	options.StateReport = stateReport
	return pcmilerRuntime{client: client, options: options, profile: profile, ready: true}
}

func (f *resolveFixture) noStoredMileage() {
	f.stored.EXPECT().
		Lookup(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("StoredMileage")).
		Maybe()
	f.buffer.EXPECT().Push(mock.Anything, mock.Anything).Return(nil).Maybe()
}

func TestResolveForShipmentRoutesPayAndBillingWithTheirOwnProfiles(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.noStoredMileage()
	pay := f.purposeProfile("Practical")
	billing := f.purposeProfile("Shortest")
	f.control.PayDistanceProfileID = pay.ID
	f.control.BillingDistanceProfileID = billing.ID

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{
		{Distance: 200, JurisdictionDistances: []pcmiler.JurisdictionDistance{
			{Country: "US", Code: "TX", Distance: 120},
			{Country: "US", Code: "OK", Distance: 80},
		}},
		{Distance: 212.25},
		{Distance: 188},
	}}
	runtimes := f.runtime(client, true)
	runtimes[distancecontrol.PurposePay] = purposeRuntime(client, pay, false)
	runtimes[distancecontrol.PurposeBilling] = purposeRuntime(client, billing, false)

	resp, err := f.service.resolveForShipment(t.Context(), f.entity, runtimes)
	require.NoError(t, err)

	require.Len(t, client.requests, 1, "every purpose rides the one PC*Miler call")
	requests := client.requests[0]
	require.Len(t, requests, 3)
	assert.Equal(t, f.profile.RoutingType, requests[0].Options.RoutingType)
	assert.True(t, requests[0].Options.StateReport, "fuel shares the loaded profile")
	assert.Equal(t, "Practical", requests[1].Options.RoutingType)
	assert.False(t, requests[1].Options.StateReport)
	assert.Equal(t, "Shortest", requests[2].Options.RoutingType)
	assert.NotEqual(t, requests[0].RouteID, requests[1].RouteID)
	assert.NotEqual(t, requests[1].RouteID, requests[2].RouteID)

	require.NotNil(t, f.move.Distance)
	assert.Equal(t, 200.0, *f.move.Distance)
	require.NotNil(t, f.move.PayDistance)
	assert.Equal(t, 212.25, *f.move.PayDistance)
	require.NotNil(t, f.move.BillingDistance)
	assert.Equal(t, 188.0, *f.move.BillingDistance)
	assert.Nil(t, f.move.FuelDistance)
	assert.Equal(t, 200.0, *f.move.FuelTaxDistance())
	require.Len(t, f.move.JurisdictionMiles, 2)
	assert.Equal(t, f.profile.ID, f.move.JurisdictionMiles[0].DistanceProfileID)

	purposes, ok := f.move.DistanceMetadata["purposeDistances"].(map[string]any)
	require.True(t, ok)
	payMeta, ok := purposes[distancecontrol.PurposePay].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, pay.ID.String(), payMeta["distanceProfileId"])
	assert.Equal(t, distancecalculation.SourcePCMiler, payMeta["source"])

	require.Len(t, resp.Moves, 1)
	require.NotNil(t, resp.Moves[0].PayDistance)
	assert.Equal(t, 212.25, *resp.Moves[0].PayDistance)
	require.NotNil(t, resp.Moves[0].BillingDistance)
	assert.Equal(t, 188.0, *resp.Moves[0].BillingDistance)
	assert.Equal(t, 200.0, resp.TotalDistance)
}

func TestResolveForShipmentSharesOneRouteBetweenPurposesOnTheSameProfile(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.noStoredMileage()
	shortest := f.purposeProfile("Shortest")
	f.control.PayDistanceProfileID = shortest.ID
	f.control.BillingDistanceProfileID = shortest.ID

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 200}, {Distance: 190}}}
	runtimes := f.runtime(client, true)
	runtimes[distancecontrol.PurposePay] = purposeRuntime(client, shortest, false)

	_, err := f.service.resolveForShipment(t.Context(), f.entity, runtimes)
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	require.Len(t, client.requests[0], 2)
	require.NotNil(t, f.move.PayDistance)
	require.NotNil(t, f.move.BillingDistance)
	assert.Equal(t, 190.0, *f.move.PayDistance)
	assert.Equal(t, 190.0, *f.move.BillingDistance)
}

func TestResolveForShipmentTakesFuelJurisdictionsFromTheFuelRoute(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.noStoredMileage()
	fuel := f.purposeProfile("Practical")
	f.control.FuelDistanceProfileID = fuel.ID

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{
		{Distance: 200},
		{Distance: 205, JurisdictionDistances: []pcmiler.JurisdictionDistance{
			{Country: "US", Code: "TX", Distance: 125},
			{Country: "US", Code: "OK", Distance: 80},
		}},
	}}
	runtimes := f.runtime(client, true)
	loaded := runtimes[distancecontrol.PurposeLoadedMove]
	loaded.options.StateReport = false
	runtimes[distancecontrol.PurposeLoadedMove] = loaded
	runtimes[distancecontrol.PurposeFuel] = purposeRuntime(client, fuel, true)

	resp, err := f.service.resolveForShipment(t.Context(), f.entity, runtimes)
	require.NoError(t, err)

	requests := client.requests[0]
	require.Len(t, requests, 2)
	assert.False(t, requests[0].Options.StateReport)
	assert.True(t, requests[1].Options.StateReport)

	assert.Equal(t, 200.0, *f.move.Distance)
	require.NotNil(t, f.move.FuelDistance)
	assert.Equal(t, 205.0, *f.move.FuelDistance)
	assert.Nil(t, f.move.PayDistance)
	assert.Equal(t, 200.0, *f.move.PaidDistance())
	require.Len(t, f.move.JurisdictionMiles, 2)
	assert.Equal(t, "TX", f.move.JurisdictionMiles[0].JurisdictionCode)
	assert.Equal(t, 125.0, f.move.JurisdictionMiles[0].Distance)
	assert.Equal(t, fuel.ID, f.move.JurisdictionMiles[0].DistanceProfileID)
	assert.True(t, f.move.JurisdictionMilesDirty)

	require.Len(t, resp.Moves, 1)
	require.Len(t, resp.Moves[0].JurisdictionMiles, 2)
	assert.Empty(t, resp.Moves[0].Warnings, "the breakdown matches the fuel route it came from")
}

func TestResolveForShipmentReadsAPurposeFromStoredMileageOfItsOwnProfile(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.buffer.EXPECT().Push(mock.Anything, mock.Anything).Return(nil).Maybe()
	billing := f.purposeProfile("Shortest")
	billing.DistanceUnits = "Kilometers"
	f.control.BillingDistanceProfileID = billing.ID

	stored := &storedmileage.StoredMileage{
		ID:                  pulid.MustNew("smg_"),
		Distance:            321.87,
		DistanceUnits:       "Kilometers",
		Provider:            "PCMiler",
		DataVersion:         "Current",
		RoutingType:         "Shortest",
		DistanceProfileID:   billing.ID,
		DistanceProfileName: billing.Name,
	}
	f.stored.EXPECT().
		Lookup(mock.Anything, mock.MatchedBy(func(req storedmileageports.StoredMileageLookupRequest) bool {
			return req.DistanceProfileID == billing.ID
		})).
		Return(stored, nil)
	f.stored.EXPECT().
		Lookup(mock.Anything, mock.MatchedBy(func(req storedmileageports.StoredMileageLookupRequest) bool {
			return req.DistanceProfileID == f.profile.ID
		})).
		Return(nil, errortypes.NewNotFoundError("StoredMileage"))
	f.stored.EXPECT().
		IncrementHit(mock.Anything, stored.ID, mock.Anything).
		Return(nil).
		Maybe()

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 210}}}
	runtimes := f.runtime(client, true)
	runtimes[distancecontrol.PurposeBilling] = purposeRuntime(client, billing, false)

	_, err := f.service.resolveForShipment(t.Context(), f.entity, runtimes)
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	require.Len(t, client.requests[0], 1, "the billing lane was already stored for its profile")
	assert.Equal(t, 210.0, *f.move.Distance)
	require.NotNil(t, f.move.BillingDistance)
	assert.Equal(t, 200.0, *f.move.BillingDistance, "321.87 km in the move's miles")
}

func TestResolveForShipmentLeavesPurposesToTheManualDistanceWhenRoutingFails(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.noStoredMileage()
	pay := f.purposeProfile("Shortest")
	f.control.PayDistanceProfileID = pay.ID
	existing, stale := 150.0, 99.0
	f.move.Distance = &existing
	f.move.PayDistance = &stale

	client := &fakeMileageClient{err: assert.AnError}
	runtimes := f.runtime(client, true)
	runtimes[distancecontrol.PurposePay] = purposeRuntime(client, pay, false)

	_, err := f.service.resolveForShipment(t.Context(), f.entity, runtimes)
	require.NoError(t, err)

	assert.Equal(t, distancecalculation.SourceManual, f.move.DistanceSource)
	assert.Nil(t, f.move.PayDistance)
	assert.Equal(t, 150.0, *f.move.PaidDistance())
}

func TestResolveForShipmentWarnsWhenAPurposeRouteComesBackEmpty(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.noStoredMileage()
	pay := f.purposeProfile("Shortest")
	f.control.PayDistanceProfileID = pay.ID

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 200}}}
	runtimes := f.runtime(client, true)
	runtimes[distancecontrol.PurposePay] = purposeRuntime(client, pay, false)

	resp, err := f.service.resolveForShipment(t.Context(), f.entity, runtimes)
	require.NoError(t, err)

	assert.Equal(t, distancecalculation.SourcePCMiler, f.move.DistanceSource)
	assert.Nil(t, f.move.PayDistance)
	require.Len(t, resp.Moves, 1)
	assert.Contains(
		t,
		resp.Moves[0].Warnings,
		"Pay distance could not be routed with its distance profile, so the move's distance stands for it",
	)
}

func TestResolveForShipmentRoutesNoPurposeThatSharesTheMoveProfile(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.noStoredMileage()
	stale := 99.0
	f.move.BillingDistance = &stale

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 200}}}
	_, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	require.Len(t, client.requests[0], 1)
	assert.Nil(t, f.move.PayDistance)
	assert.Nil(t, f.move.BillingDistance)
	assert.Nil(t, f.move.FuelDistance)
	_, ok := f.move.DistanceMetadata["purposeDistances"]
	assert.False(t, ok)
}

func TestPurposeCapturesJurisdictions(t *testing.T) {
	t.Parallel()

	loaded := pulid.MustNew("dp_")
	empty := pulid.MustNew("dp_")
	fuel := pulid.MustNew("dp_")
	control := &distancecontrol.DistanceControl{
		LoadedMoveDistanceProfileID: loaded,
		EmptyMoveDistanceProfileID:  empty,
		FuelDistanceProfileID:       loaded,
	}

	assert.True(t, purposeCapturesJurisdictions(control, distancecontrol.PurposeLoadedMove))
	assert.False(t, purposeCapturesJurisdictions(control, distancecontrol.PurposeEmptyMove))
	assert.True(t, purposeCapturesJurisdictions(control, distancecontrol.PurposeFuel))
	assert.False(t, purposeCapturesJurisdictions(control, distancecontrol.PurposePay))

	control.FuelDistanceProfileID = fuel
	assert.False(t, purposeCapturesJurisdictions(control, distancecontrol.PurposeLoadedMove))
	assert.True(t, purposeCapturesJurisdictions(control, distancecontrol.PurposeFuel))
}

func TestSeparateFuelDistance(t *testing.T) {
	t.Parallel()

	loaded := pulid.MustNew("dp_")
	control := &distancecontrol.DistanceControl{
		LoadedMoveDistanceProfileID: loaded,
		FuelDistanceProfileID:       loaded,
	}
	move := &shipment.ShipmentMove{
		Loaded:         true,
		DistanceSource: distancecalculation.SourcePCMiler,
		DistanceUnits:  "Miles",
	}

	_, separate := separateFuelDistance(control, move, 300, "Miles")
	assert.False(t, separate, "fuel shares the move's profile")

	control.FuelDistanceProfileID = pulid.MustNew("dp_")
	distance, separate := separateFuelDistance(control, move, 482.8032, "Kilometers")
	assert.True(t, separate)
	assert.Equal(t, 300.0, distance)

	move.DistanceSource = distancecalculation.SourceManual
	_, separate = separateFuelDistance(control, move, 300, "Miles")
	assert.False(t, separate, "a distance entered by hand stands for fuel")
}

func TestSeparatePurposeGroups(t *testing.T) {
	t.Parallel()

	loaded := pulid.MustNew("dp_")
	shortest := pulid.MustNew("dp_")
	control := &distancecontrol.DistanceControl{
		LoadedMoveDistanceProfileID: loaded,
		EmptyMoveDistanceProfileID:  shortest,
		PayDistanceProfileID:        shortest,
		BillingDistanceProfileID:    shortest,
		FuelDistanceProfileID:       loaded,
	}

	groups := separatePurposeGroups(control, distancecontrol.PurposeLoadedMove)
	require.Len(t, groups, 1)
	assert.Equal(t, shortest, groups[0].profileID)
	assert.Equal(
		t,
		[]string{distancecontrol.PurposePay, distancecontrol.PurposeBilling},
		groups[0].purposes,
	)

	groups = separatePurposeGroups(control, distancecontrol.PurposeEmptyMove)
	require.Len(t, groups, 1)
	assert.Equal(t, []string{distancecontrol.PurposeFuel}, groups[0].purposes)
	assert.Equal(t, distancecontrol.PurposeFuel, groups[0].lead())
}

func TestRecalculateMoveJurisdictionMilesRoutesWithTheFuelProfile(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.buffer.EXPECT().Push(mock.Anything, mock.Anything).Return(nil).Maybe()
	fuel := f.purposeProfile("Practical")
	f.control.FuelDistanceProfileID = fuel.ID
	routed := 200.0
	f.move.Distance = &routed
	f.move.DistanceUnits = "Miles"
	f.move.DistanceSource = distancecalculation.SourcePCMiler
	f.move.Status = shipmentvocab.MoveStatusCompleted

	moves := shipmentmoveportsmock.NewMockShipmentMoveRepository(t)
	moves.EXPECT().GetByID(mock.Anything, mock.Anything).Return(f.move, nil)
	shipments := shipmentportsmock.NewMockShipmentRepository(t)
	shipments.EXPECT().GetByID(mock.Anything, mock.Anything).Return(f.entity, nil)
	rows := shipmentdistanceportsmock.NewMockShipmentMoveJurisdictionMileRepository(t)
	rows.EXPECT().ReplaceForMove(mock.Anything, f.move).Return(nil)
	calcRepo := distancecalculationportsmock.NewMockDistanceCalculationRepository(t)
	var persisted *float64
	calcRepo.EXPECT().
		UpdateMoveDistance(mock.Anything, f.move).
		Run(func(_ context.Context, move *shipment.ShipmentMove) { persisted = move.FuelDistance }).
		Return(nil)
	var run *distancecalculation.Run
	calcRepo.EXPECT().
		CreateRun(mock.Anything, mock.Anything).
		Run(func(_ context.Context, entity *distancecalculation.Run) { run = entity }).
		Return(nil)
	f.service.shipmentMoveRepo = moves
	f.service.shipmentRepo = shipments
	f.service.jurisdictionMileRepo = rows
	f.service.distanceCalcRepo = calcRepo

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{
		Distance: 205,
		JurisdictionDistances: []pcmiler.JurisdictionDistance{
			{Country: "US", Code: "TX", Distance: 125},
			{Country: "US", Code: "OK", Distance: 80},
		},
	}}}
	runtimes := map[string]pcmilerRuntime{
		distancecontrol.PurposeFuel: purposeRuntime(client, fuel, true),
	}

	got, err := f.service.recalculateMoveJurisdictionMiles(
		t.Context(),
		&shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: f.entity.OrganizationID,
				BuID:  f.entity.BusinessUnitID,
			},
			ShipmentMoveID: f.move.ID,
		},
		runtimes,
	)
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	assert.True(t, client.requests[0][0].Options.StateReport)
	require.Len(t, got, 2)
	assert.Equal(t, fuel.ID, got[0].DistanceProfileID)
	require.NotNil(t, persisted)
	assert.Equal(t, 205.0, *persisted)
	assert.Equal(t, 200.0, *f.move.Distance, "the routed distance is untouched")
	require.NotNil(t, run)
	assert.Equal(t, fuel.ID.String(), run.RequestSummary["distance_profile_id"])
	assert.Equal(
		t,
		JurisdictionAttributionAttributed,
		run.ResponseSummary["jurisdictionAttribution"],
	)
}

func TestStoreFuelDistanceClearsAStaleFuelDistance(t *testing.T) {
	t.Parallel()

	loaded := pulid.MustNew("dp_")
	control := &distancecontrol.DistanceControl{
		LoadedMoveDistanceProfileID: loaded,
		FuelDistanceProfileID:       loaded,
	}
	stale := 250.0
	move := &shipment.ShipmentMove{
		Loaded:         true,
		DistanceSource: distancecalculation.SourcePCMiler,
		DistanceUnits:  "Miles",
		FuelDistance:   &stale,
	}
	calcRepo := distancecalculationportsmock.NewMockDistanceCalculationRepository(t)
	calcRepo.EXPECT().UpdateMoveDistance(mock.Anything, move).Return(nil).Once()
	svc := &Service{distanceCalcRepo: calcRepo}

	require.NoError(t, svc.storeFuelDistance(
		t.Context(),
		control,
		move,
		200,
		pcmiler.RouteOptions{DistanceUnits: "Miles"},
	))
	assert.Nil(t, move.FuelDistance)

	require.NoError(t, svc.storeFuelDistance(
		t.Context(),
		control,
		move,
		200,
		pcmiler.RouteOptions{DistanceUnits: "Miles"},
	), "nothing to write when fuel shares the move's profile and holds no distance")
}
