package distancecalculationservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/core/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMileageClient struct {
	requests [][]pcmiler.RouteRequest
	results  []pcmiler.RouteMileage
	err      error
}

func (f *fakeMileageClient) Mileage(
	_ context.Context,
	routes []pcmiler.RouteRequest,
) ([]pcmiler.RouteMileage, error) {
	f.requests = append(f.requests, routes)
	if f.err != nil {
		return nil, f.err
	}
	results := make([]pcmiler.RouteMileage, 0, len(routes))
	for idx, route := range routes {
		if idx >= len(f.results) {
			break
		}
		result := f.results[idx]
		result.RouteID = route.RouteID
		results = append(results, result)
	}
	return results, nil
}

func testLocation(city, postalCode, state, countryISO3 string) *location.Location {
	return &location.Location{
		ID:         pulid.MustNew("loc_"),
		City:       city,
		PostalCode: postalCode,
		State: &usstate.UsState{
			ID:           pulid.MustNew("us_"),
			Abbreviation: state,
			CountryIso3:  countryISO3,
		},
	}
}

func testMove(orgID, buID pulid.ID, loaded bool) *shipment.ShipmentMove {
	origin := testLocation("Dallas", "75201", "TX", "USA")
	destination := testLocation("Oklahoma City", "73102", "OK", "USA")
	return &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		Loaded:         loaded,
		Stops: []*shipment.Stop{
			{ID: pulid.MustNew("stp_"), LocationID: origin.ID, Sequence: 0, Location: origin},
			{
				ID:         pulid.MustNew("stp_"),
				LocationID: destination.ID,
				Sequence:   1,
				Location:   destination,
			},
		},
	}
}

func TestApplyMoveDistanceBuildsJurisdictionRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		loaded bool
	}{
		{name: "loaded move", loaded: true},
		{name: "empty move", loaded: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			move := testMove(pulid.MustNew("org_"), pulid.MustNew("bu_"), tt.loaded)
			profileID := pulid.MustNew("dp_")
			warnings := applyMoveDistance(moveDistanceParams{
				move:         move,
				distance:     200,
				source:       distancecalculation.SourcePCMiler,
				provider:     "PCMiler",
				dataVersion:  "Current",
				distanceUnit: "Miles",
				profileID:    profileID.String(),
				calculatedAt: 1_700_000_000,
				jurisdictions: []pcmiler.JurisdictionDistance{
					{Country: "US", Code: "TX", Distance: 120.4, Toll: 3.5},
					{Country: "US", Code: "OK", Distance: 79.6},
				},
			})

			assert.Empty(t, warnings)
			assert.True(t, move.JurisdictionMilesDirty)
			require.Len(t, move.JurisdictionMiles, 2)

			first := move.JurisdictionMiles[0]
			assert.Equal(t, "US", first.CountryCode)
			assert.Equal(t, "TX", first.JurisdictionCode)
			assert.Equal(t, 0, first.Sequence)
			assert.InDelta(t, 120.4, first.Distance, 0.001)
			assert.Equal(t, "Miles", first.DistanceUnits)
			assert.Equal(t, tt.loaded, first.Loaded)
			assert.Equal(t, shipment.JurisdictionMileSourceRouteCalculation, first.Source)
			assert.Equal(t, "PCMiler", first.Provider)
			assert.Equal(t, "Current", first.DataVersion)
			assert.Equal(t, profileID, first.DistanceProfileID)
			assert.Equal(t, int64(1_700_000_000), first.CalculatedAt)
			assert.Equal(t, move.ID, first.ShipmentMoveID)
			assert.Equal(t, move.ShipmentID, first.ShipmentID)
			require.NotNil(t, first.TollDistance)
			assert.InDelta(t, 3.5, *first.TollDistance, 0.001)
			assert.Nil(t, first.FerryDistance)

			second := move.JurisdictionMiles[1]
			assert.Equal(t, "OK", second.JurisdictionCode)
			assert.Equal(t, 1, second.Sequence)
			assert.Equal(t, tt.loaded, second.Loaded)
		})
	}
}

func TestApplyMoveDistanceWarnsOnMismatchedBreakdown(t *testing.T) {
	t.Parallel()

	move := testMove(pulid.MustNew("org_"), pulid.MustNew("bu_"), true)
	warnings := applyMoveDistance(moveDistanceParams{
		move:         move,
		distance:     200,
		source:       distancecalculation.SourcePCMiler,
		distanceUnit: "Miles",
		metadata:     map[string]any{"warnings": []string{"provider warning"}},
		jurisdictions: []pcmiler.JurisdictionDistance{
			{Country: "US", Code: "TX", Distance: 150},
			{Country: "US", Code: "OK", Distance: 40},
		},
	})

	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "190.00 Miles")
	assert.Contains(t, warnings[0], "200.00 Miles")
	assert.Len(t, move.JurisdictionMiles, 2)
	assert.Equal(t, []string{"provider warning", warnings[0]}, move.DistanceMetadata["warnings"])
}

func TestJurisdictionToleranceWithKilometers(t *testing.T) {
	t.Parallel()

	stored := []storedmileage.JurisdictionDistance{
		{Country: "US", Code: "TX", Distance: 62.137},
		{Country: "US", Code: "OK", Distance: 37.863},
	}
	converted := convertStoredJurisdictions(stored, "Miles", "Kilometers")
	require.Len(t, converted, 2)
	assert.InDelta(t, 100, converted[0].Distance, 0.01)
	assert.InDelta(t, 60.93, converted[1].Distance, 0.01)

	move := testMove(pulid.MustNew("org_"), pulid.MustNew("bu_"), true)
	warnings := applyMoveDistance(moveDistanceParams{
		move:          move,
		distance:      160.93,
		source:        distancecalculation.SourceStoredMileage,
		distanceUnit:  "Kilometers",
		jurisdictions: pcmilerJurisdictionsFromStored(converted),
	})
	assert.Empty(t, warnings)
	require.Len(t, move.JurisdictionMiles, 2)
	assert.Equal(t, "Kilometers", move.JurisdictionMiles[0].DistanceUnits)
	assert.InDelta(t, 62.14, move.JurisdictionMiles[0].DistanceInMiles(), 0.01)

	attribution, sum := jurisdictionAttribution(100, "Miles", move.JurisdictionMiles)
	assert.Equal(t, JurisdictionAttributionAttributed, attribution)
	assert.InDelta(t, 100, sum, 0.01)

	assert.True(t, withinJurisdictionTolerance(100, 100.5))
	assert.False(t, withinJurisdictionTolerance(100, 100.51))
	assert.True(t, withinJurisdictionTolerance(1000, 1004.9))
	assert.False(t, withinJurisdictionTolerance(1000, 1005.1))
}

func TestApplyManualDistanceClearsJurisdictionRows(t *testing.T) {
	t.Parallel()

	distance := 42.0
	move := &shipment.ShipmentMove{
		ID:       pulid.MustNew("sm_"),
		Distance: &distance,
		JurisdictionMiles: []*shipment.ShipmentMoveJurisdictionMile{
			{CountryCode: "US", JurisdictionCode: "TX", Distance: 42},
		},
	}

	got := applyManualDistance(move, "*|", 1_700_000_000)

	assert.Equal(t, 42.0, got)
	assert.Empty(t, move.JurisdictionMiles)
	assert.True(t, move.JurisdictionMilesDirty)
	assert.Equal(t, distancecalculation.SourceManual, move.DistanceSource)
}

func TestLocationToPCMilerStopCountry(t *testing.T) {
	t.Parallel()

	options := distanceprofile.NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_")).RouteOptions()
	tests := []struct {
		iso3 string
		want string
	}{
		{iso3: "USA", want: "US"},
		{iso3: "CAN", want: "CA"},
		{iso3: "MEX", want: "MX"},
		{iso3: "GBR", want: "US"},
		{iso3: "", want: "US"},
	}
	for _, tt := range tests {
		stop := locationToPCMilerStop(testLocation("City", "12345", "ZZ", tt.iso3), options)
		assert.Equal(t, tt.want, stop.Country, tt.iso3)
		assert.Equal(t, "ZZ", stop.State)
	}

	noState := locationToPCMilerStop(&location.Location{City: "City", PostalCode: "12345"}, options)
	assert.Equal(t, "US", noState.Country)
}

func TestUseStoredMileage(t *testing.T) {
	t.Parallel()

	withBreakdown := &storedmileage.StoredMileage{
		JurisdictionDistances: []storedmileage.JurisdictionDistance{{Country: "US", Code: "TX"}},
	}
	withoutBreakdown := &storedmileage.StoredMileage{}
	ready := pcmilerRuntime{ready: true, options: pcmiler.RouteOptions{StateReport: true}}
	readyNoReport := pcmilerRuntime{ready: true}
	notReady := pcmilerRuntime{options: pcmiler.RouteOptions{StateReport: true}}

	assert.True(t, useStoredMileage(withBreakdown, ready))
	assert.False(t, useStoredMileage(withoutBreakdown, ready))
	assert.True(t, useStoredMileage(withoutBreakdown, readyNoReport))
	assert.True(t, useStoredMileage(withoutBreakdown, notReady))
	assert.False(t, useStoredMileage(nil, ready))
}

type resolveFixture struct {
	service  *Service
	control  *distancecontrol.DistanceControl
	profile  *distanceprofile.DistanceProfile
	stored   *mocks.MockStoredMileageRepository
	buffer   *mocks.MockStoredMileageBufferRepository
	override *mocks.MockDistanceOverrideRepository
	entity   *shipment.Shipment
	move     *shipment.ShipmentMove
}

func newResolveFixture(t *testing.T) *resolveFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	profile := distanceprofile.NewDefault(orgID, buID)
	profile.ID = pulid.MustNew("dp_")
	control := distancecontrol.NewDefault(orgID, buID, profile.ID, profile.ID)
	control.CaptureJurisdictionMiles = true

	controlRepo := mocks.NewMockDistanceControlRepository(t)
	controlRepo.EXPECT().EnsureDefault(mock.Anything, mock.Anything).Return(control, nil)
	overrideRepo := mocks.NewMockDistanceOverrideRepository(t)
	overrideRepo.EXPECT().
		GetByRouteSignature(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Distance override"))
	storedRepo := mocks.NewMockStoredMileageRepository(t)
	bufferRepo := mocks.NewMockStoredMileageBufferRepository(t)

	move := testMove(orgID, buID, true)
	entity := &shipment.Shipment{
		ID:             move.ShipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Moves:          []*shipment.ShipmentMove{move},
	}

	return &resolveFixture{
		service: &Service{
			l:                    zap.NewNop(),
			distanceOverrideRepo: overrideRepo,
			distanceControlRepo:  controlRepo,
			storedMileageRepo:    storedRepo,
			storedMileageBuffer:  bufferRepo,
		},
		control:  control,
		profile:  profile,
		stored:   storedRepo,
		buffer:   bufferRepo,
		override: overrideRepo,
		entity:   entity,
		move:     move,
	}
}

func (f *resolveFixture) storedWithoutBreakdown() *storedmileage.StoredMileage {
	return &storedmileage.StoredMileage{
		ID:                  pulid.MustNew("smg_"),
		OrganizationID:      f.entity.OrganizationID,
		BusinessUnitID:      f.entity.BusinessUnitID,
		Distance:            180,
		DistanceUnits:       "Miles",
		Provider:            "PCMiler",
		RouteSignature:      "cached",
		DataVersion:         "Current",
		RoutingType:         "Practical",
		DistanceProfileID:   f.profile.ID,
		DistanceProfileName: f.profile.Name,
	}
}

func (f *resolveFixture) runtime(client mileageClient, ready bool) map[string]pcmilerRuntime {
	options := f.profile.RouteOptions()
	options.StateReport = true
	return map[string]pcmilerRuntime{
		distancecontrol.PurposeLoadedMove: {
			client:  client,
			options: options,
			profile: f.profile,
			ready:   ready,
		},
	}
}

func TestResolveForShipmentRoutesAgainWhenCacheLacksBreakdown(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.stored.EXPECT().Lookup(mock.Anything, mock.Anything).Return(f.storedWithoutBreakdown(), nil)

	var pushed *storedmileage.StoredMileage
	f.buffer.EXPECT().
		Push(mock.Anything, mock.Anything).
		Run(func(_ context.Context, candidate *storedmileage.StoredMileage) {
			pushed = candidate
		}).
		Return(nil)

	client := &fakeMileageClient{
		results: []pcmiler.RouteMileage{{
			Distance:    200,
			DataVersion: "Current",
			JurisdictionDistances: []pcmiler.JurisdictionDistance{
				{Country: "US", Code: "TX", Distance: 120},
				{Country: "US", Code: "OK", Distance: 80},
			},
		}},
	}

	resp, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	require.Len(t, client.requests[0], 1)
	request := client.requests[0][0]
	assert.True(t, request.Options.StateReport)
	require.Len(t, request.Stops, 2)
	assert.Equal(t, "US", request.Stops[0].Country)
	assert.Equal(t, "TX", request.Stops[0].State)

	assert.Equal(t, distancecalculation.SourcePCMiler, f.move.DistanceSource)
	require.NotNil(t, f.move.Distance)
	assert.Equal(t, 200.0, *f.move.Distance)
	assert.True(t, f.move.JurisdictionMilesDirty)
	require.Len(t, f.move.JurisdictionMiles, 2)
	assert.Equal(t, "TX", f.move.JurisdictionMiles[0].JurisdictionCode)
	assert.True(t, f.move.JurisdictionMiles[0].Loaded)

	require.Len(t, resp.Moves, 1)
	require.Len(t, resp.Moves[0].JurisdictionMiles, 2)
	assert.Equal(t, "OK", resp.Moves[0].JurisdictionMiles[1].JurisdictionCode)
	assert.Equal(t, 200.0, resp.TotalDistance)

	require.NotNil(t, pushed)
	require.Len(t, pushed.JurisdictionDistances, 2)
	assert.Equal(t, "TX", pushed.JurisdictionDistances[0].Code)
	f.stored.AssertNotCalled(t, "IncrementHit", mock.Anything, mock.Anything, mock.Anything)
}

func TestResolveForShipmentUsesCacheWithoutRowsWhenProviderNotReady(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	stored := f.storedWithoutBreakdown()
	f.stored.EXPECT().Lookup(mock.Anything, mock.Anything).Return(stored, nil)
	f.stored.EXPECT().
		IncrementHit(mock.Anything, stored.ID, mock.Anything).
		Return(nil).
		Maybe()
	f.move.JurisdictionMiles = []*shipment.ShipmentMoveJurisdictionMile{
		{CountryCode: "US", JurisdictionCode: "TX", Distance: 180},
	}

	client := &fakeMileageClient{}
	resp, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, false))
	require.NoError(t, err)

	assert.Empty(t, client.requests)
	assert.Equal(t, distancecalculation.SourceStoredMileage, f.move.DistanceSource)
	require.NotNil(t, f.move.Distance)
	assert.Equal(t, 180.0, *f.move.Distance)
	assert.Empty(t, f.move.JurisdictionMiles)
	assert.True(t, f.move.JurisdictionMilesDirty)
	require.Len(t, resp.Moves, 1)
	assert.Empty(t, resp.Moves[0].JurisdictionMiles)
	assert.Equal(t, 180.0, resp.TotalDistance)
}

func TestResolveForShipmentUsesCachedBreakdown(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	stored := f.storedWithoutBreakdown()
	stored.JurisdictionDistances = []storedmileage.JurisdictionDistance{
		{Country: "US", Code: "TX", Distance: 100},
		{Country: "US", Code: "OK", Distance: 80},
	}
	f.stored.EXPECT().Lookup(mock.Anything, mock.Anything).Return(stored, nil)
	f.stored.EXPECT().
		IncrementHit(mock.Anything, stored.ID, mock.Anything).
		Return(nil).
		Maybe()

	client := &fakeMileageClient{}
	_, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	assert.Empty(t, client.requests)
	assert.Equal(t, distancecalculation.SourceStoredMileage, f.move.DistanceSource)
	require.Len(t, f.move.JurisdictionMiles, 2)
	assert.Equal(t, "OK", f.move.JurisdictionMiles[1].JurisdictionCode)
	assert.True(t, f.move.JurisdictionMilesDirty)
}

func TestResolveForShipmentLeavesRowsEmptyWhenProviderFails(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	f.stored.EXPECT().
		Lookup(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("StoredMileage"))
	existing := 150.0
	f.move.Distance = &existing

	client := &fakeMileageClient{err: assert.AnError}
	resp, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	assert.Equal(t, distancecalculation.SourceManual, f.move.DistanceSource)
	assert.Equal(t, 150.0, *f.move.Distance)
	assert.Empty(t, f.move.JurisdictionMiles)
	assert.True(t, f.move.JurisdictionMilesDirty)
	assert.Equal(t, 150.0, resp.TotalDistance)
}

func TestJurisdictionAttributionStates(t *testing.T) {
	t.Parallel()

	attribution, sum := jurisdictionAttribution(100, "Miles", nil)
	assert.Equal(t, JurisdictionAttributionUnattributed, attribution)
	assert.Equal(t, 0.0, sum)

	rows := []*shipment.ShipmentMoveJurisdictionMile{
		{Distance: 60, DistanceUnits: "Miles"},
		{Distance: 30, DistanceUnits: "Miles"},
	}
	attribution, sum = jurisdictionAttribution(100, "Miles", rows)
	assert.Equal(t, JurisdictionAttributionMismatch, attribution)
	assert.Equal(t, 90.0, sum)

	rows = append(rows, &shipment.ShipmentMoveJurisdictionMile{Distance: 10, DistanceUnits: "Miles"})
	attribution, _ = jurisdictionAttribution(100, "Miles", rows)
	assert.Equal(t, JurisdictionAttributionAttributed, attribution)
}
