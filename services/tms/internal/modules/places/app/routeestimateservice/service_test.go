package routeestimateservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeMileage struct {
	calls [][]pcmiler.RouteRequest
	miles float64
	drive int64
	err   error
}

func (f *fakeMileage) Mileage(
	_ context.Context,
	routes []pcmiler.RouteRequest,
) ([]pcmiler.RouteMileage, error) {
	f.calls = append(f.calls, routes)
	if f.err != nil {
		return nil, f.err
	}
	out := make([]pcmiler.RouteMileage, 0, len(routes))
	for _, route := range routes {
		out = append(out, pcmiler.RouteMileage{
			RouteID:      route.RouteID,
			Distance:     f.miles,
			DriveSeconds: f.drive,
		})
	}
	return out, nil
}

var (
	dallas  = routeestimateports.Point{Latitude: 32.7767, Longitude: -96.797}
	houston = routeestimateports.Point{Latitude: 29.7604, Longitude: -95.3698}
	austin  = routeestimateports.Point{Latitude: 30.2672, Longitude: -97.7431}
)

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func withRuntime(runtime tenantRuntime, resolves *int) *service {
	return newService(zap.NewNop(), func(context.Context, pagination.TenantInfo) tenantRuntime {
		*resolves++
		return runtime
	})
}

func TestEstimateLegsUsesThePlanningSpeedWithoutPCMiler(t *testing.T) {
	t.Parallel()

	resolves := 0
	svc := withRuntime(tenantRuntime{}, &resolves)

	got, err := svc.EstimateLegs(t.Context(), tenant(), []routeestimateports.Leg{
		{From: dallas, To: houston},
	})

	require.NoError(t, err)
	require.Len(t, got, 1)
	miles := geoutils.PlanningRoadMiles(
		dallas.Latitude,
		dallas.Longitude,
		houston.Latitude,
		houston.Longitude,
	)
	assert.InDelta(t, miles, got[0].Miles, 0.001)
	assert.Equal(t, geoutils.PlanningDriveSeconds(miles), got[0].DriveSeconds)
	assert.False(t, got[0].Routed)
}

func TestEstimateLegsRoutesOnceAndCachesTheLeg(t *testing.T) {
	t.Parallel()

	client := &fakeMileage{miles: 239.4, drive: 3*3600 + 55*60}
	resolves := 0
	svc := withRuntime(tenantRuntime{client: client, ready: true, scope: "prof:1"}, &resolves)
	who := tenant()
	legs := []routeestimateports.Leg{
		{From: dallas, To: houston},
		{From: dallas, To: houston},
		{From: houston, To: austin},
	}

	first, err := svc.EstimateLegs(t.Context(), who, legs)
	require.NoError(t, err)
	require.Len(t, client.calls, 1)
	assert.Len(t, client.calls[0], 2, "the repeated leg is routed once")
	for _, leg := range first {
		assert.True(t, leg.Routed)
		assert.InDelta(t, 239.4, leg.Miles, 0.001)
		assert.Equal(t, int64(3*3600+55*60), leg.DriveSeconds)
	}

	second, err := svc.EstimateLegs(t.Context(), who, legs[:1])
	require.NoError(t, err)
	assert.Len(t, client.calls, 1, "a cached leg is not routed again")
	assert.Equal(t, first[0], second[0])
	assert.Equal(t, 1, resolves, "the tenant's runtime is resolved once")
}

func TestEstimateLegsFallsBackWhenRoutingFails(t *testing.T) {
	t.Parallel()

	client := &fakeMileage{err: errors.New("503")}
	resolves := 0
	svc := withRuntime(tenantRuntime{client: client, ready: true}, &resolves)

	got, err := svc.EstimateLegs(t.Context(), tenant(), []routeestimateports.Leg{
		{From: dallas, To: houston},
	})

	require.NoError(t, err)
	assert.False(t, got[0].Routed)
	assert.Positive(t, got[0].DriveSeconds)
}

func TestEstimateLegsFillsMissingRoutedHoursFromTheDistance(t *testing.T) {
	t.Parallel()

	client := &fakeMileage{miles: 100}
	resolves := 0
	svc := withRuntime(tenantRuntime{client: client, ready: true}, &resolves)

	got, err := svc.EstimateLegs(t.Context(), tenant(), []routeestimateports.Leg{
		{From: dallas, To: houston},
	})

	require.NoError(t, err)
	assert.True(t, got[0].Routed)
	assert.Equal(t, geoutils.PlanningDriveSeconds(100), got[0].DriveSeconds)
}

func TestEstimateLegsSkipsAZeroLengthLeg(t *testing.T) {
	t.Parallel()

	client := &fakeMileage{miles: 50}
	resolves := 0
	svc := withRuntime(tenantRuntime{client: client, ready: true}, &resolves)

	got, err := svc.EstimateLegs(t.Context(), tenant(), []routeestimateports.Leg{
		{From: dallas, To: dallas},
	})

	require.NoError(t, err)
	assert.Zero(t, got[0].DriveSeconds)
	assert.Empty(t, client.calls)
}

func TestEstimateLegsRefusesAnOversizedBatch(t *testing.T) {
	t.Parallel()

	resolves := 0
	svc := withRuntime(tenantRuntime{}, &resolves)

	_, err := svc.EstimateLegs(
		t.Context(),
		tenant(),
		make([]routeestimateports.Leg, routeestimateports.MaxLegsPerEstimate+1),
	)
	require.Error(t, err)
}
