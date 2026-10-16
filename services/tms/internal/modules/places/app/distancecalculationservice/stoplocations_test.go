package distancecalculationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/usstateportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A shipment being created arrives as the request sent it: each stop names its
// location by ID and carries no location record. PC*MILER is asked by city, state
// and postal code, so those records have to be read before the move is routed.

func storedMileageMiss(f *resolveFixture) {
	f.stored.EXPECT().
		Lookup(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("StoredMileage"))
}

func savedLocation(city, postalCode string, state *usstate.UsState) *location.Location {
	return &location.Location{
		ID:         pulid.MustNew("loc_"),
		City:       city,
		PostalCode: postalCode,
		StateID:    state.ID,
	}
}

func (f *resolveFixture) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: f.entity.OrganizationID, BuID: f.entity.BusinessUnitID}
}

func TestResolveForShipmentRoutesStopsThatNameOnlyALocationID(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	storedMileageMiss(f)
	f.buffer.EXPECT().Push(mock.Anything, mock.Anything).Return(nil).Maybe()

	texas := &usstate.UsState{ID: pulid.MustNew("us_"), Abbreviation: "TX", CountryIso3: "USA"}
	oklahoma := &usstate.UsState{ID: pulid.MustNew("us_"), Abbreviation: "OK", CountryIso3: "USA"}
	origin := savedLocation("Dallas", "75201", texas)
	destination := savedLocation("Oklahoma City", "73102", oklahoma)

	f.move.Stops[0].LocationID = origin.ID
	f.move.Stops[0].Location = nil
	f.move.Stops[1].LocationID = destination.ID
	f.move.Stops[1].Location = nil

	locations := locationportsmock.NewMockLocationRepository(t)
	locations.EXPECT().
		GetByIDs(mock.Anything, locationports.GetLocationsByIDsRequest{
			TenantInfo:  f.tenant(),
			LocationIDs: []pulid.ID{origin.ID, destination.ID},
		}).
		Return([]*location.Location{destination, origin}, nil).
		Once()
	states := usstateportsmock.NewMockUsStateRepository(t)
	states.EXPECT().
		GetByIDs(mock.Anything, []pulid.ID{texas.ID, oklahoma.ID}).
		Return([]*usstate.UsState{oklahoma, texas}, nil).
		Once()
	f.service.locationRepo = locations
	f.service.stateRepo = states

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 206}}}
	resp, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	require.Len(t, client.requests[0], 1)
	stops := client.requests[0][0].Stops
	require.Len(t, stops, 2)
	assert.Equal(t, "Dallas", stops[0].City)
	assert.Equal(t, "TX", stops[0].State)
	assert.Equal(t, "75201", stops[0].PostalCode)
	assert.Equal(t, "Oklahoma City", stops[1].City)
	assert.Equal(t, "OK", stops[1].State)

	assert.Equal(t, distancecalculation.SourcePCMiler, f.move.DistanceSource)
	require.NotNil(t, f.move.Distance)
	assert.Equal(t, 206.0, *f.move.Distance)
	assert.Equal(t, 206.0, resp.TotalDistance)
	assert.Same(t, origin, f.move.Stops[0].Location)
	assert.Same(t, texas, f.move.Stops[0].Location.State)
}

func TestResolveForShipmentRereadsAStopLocationThatNoLongerMatchesItsID(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	storedMileageMiss(f)
	f.buffer.EXPECT().Push(mock.Anything, mock.Anything).Return(nil).Maybe()

	oklahoma := &usstate.UsState{ID: pulid.MustNew("us_"), Abbreviation: "OK", CountryIso3: "USA"}
	tulsa := savedLocation("Tulsa", "74103", oklahoma)
	f.move.Stops[1].LocationID = tulsa.ID

	locations := locationportsmock.NewMockLocationRepository(t)
	locations.EXPECT().
		GetByIDs(mock.Anything, locationports.GetLocationsByIDsRequest{
			TenantInfo:  f.tenant(),
			LocationIDs: []pulid.ID{tulsa.ID},
		}).
		Return([]*location.Location{tulsa}, nil).
		Once()
	states := usstateportsmock.NewMockUsStateRepository(t)
	states.EXPECT().
		GetByIDs(mock.Anything, []pulid.ID{oklahoma.ID}).
		Return([]*usstate.UsState{oklahoma}, nil).
		Once()
	f.service.locationRepo = locations
	f.service.stateRepo = states

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 257}}}
	_, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	stops := client.requests[0][0].Stops
	assert.Equal(t, "Dallas", stops[0].City)
	assert.Equal(t, "Tulsa", stops[1].City)
	assert.Equal(t, "OK", stops[1].State)
	assert.Equal(t, 257.0, *f.move.Distance)
}

func TestResolveForShipmentReadsNoLocationsWhenEveryStopCarriesItsOwn(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	storedMileageMiss(f)
	f.buffer.EXPECT().Push(mock.Anything, mock.Anything).Return(nil).Maybe()
	f.service.locationRepo = locationportsmock.NewMockLocationRepository(t)
	f.service.stateRepo = usstateportsmock.NewMockUsStateRepository(t)

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 206}}}
	_, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	require.Len(t, client.requests, 1)
	assert.Equal(t, 206.0, *f.move.Distance)
}

func TestResolveForShipmentKeepsTheManualDistanceWhenAStopLocationIsNotFound(t *testing.T) {
	t.Parallel()

	f := newResolveFixture(t)
	storedMileageMiss(f)
	manual := 190.0
	f.move.Distance = &manual

	missing := pulid.MustNew("loc_")
	f.move.Stops[1].LocationID = missing
	f.move.Stops[1].Location = nil

	locations := locationportsmock.NewMockLocationRepository(t)
	locations.EXPECT().
		GetByIDs(mock.Anything, locationports.GetLocationsByIDsRequest{
			TenantInfo:  f.tenant(),
			LocationIDs: []pulid.ID{missing},
		}).
		Return([]*location.Location{}, nil).
		Once()
	f.service.locationRepo = locations
	f.service.stateRepo = usstateportsmock.NewMockUsStateRepository(t)

	client := &fakeMileageClient{results: []pcmiler.RouteMileage{{Distance: 999}}}
	resp, err := f.service.resolveForShipment(t.Context(), f.entity, f.runtime(client, true))
	require.NoError(t, err)

	assert.Empty(t, client.requests)
	assert.Nil(t, f.move.Stops[1].Location)
	assert.Equal(t, distancecalculation.SourceManual, f.move.DistanceSource)
	assert.Equal(t, 190.0, *f.move.Distance)
	assert.Equal(t, 190.0, resp.TotalDistance)
}

func TestResolveForShipmentFailsWhenStopLocationsCannotBeRead(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	move := testMove(orgID, buID, true)
	move.Stops[0].Location = nil
	entity := &shipment.Shipment{
		ID:             move.ShipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Moves:          []*shipment.ShipmentMove{move},
	}

	locations := locationportsmock.NewMockLocationRepository(t)
	locations.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return(nil, assert.AnError).
		Once()
	svc := &Service{
		l:            zap.NewNop(),
		locationRepo: locations,
		stateRepo:    usstateportsmock.NewMockUsStateRepository(t),
	}

	resp, err := svc.ResolveForShipment(t.Context(), entity)

	require.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, resp)
	assert.Nil(t, move.Distance)
}
