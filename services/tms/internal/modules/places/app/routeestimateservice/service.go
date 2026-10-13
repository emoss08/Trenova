package routeestimateservice

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/places/app/pcmilerruntime"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/modules/places/ports/distancecontrolports"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/ttlcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	routedLegTTL      = 6 * time.Hour
	routedLegCapacity = 50_000
	runtimeTTL        = 5 * time.Minute
	runtimeCapacity   = 5_000
	coordinatePlaces  = 3
)

type mileageClient interface {
	Mileage(ctx context.Context, routes []pcmiler.RouteRequest) ([]pcmiler.RouteMileage, error)
}

type tenantRuntime struct {
	client  mileageClient
	options pcmiler.RouteOptions
	scope   string
	ready   bool
}

type Params struct {
	fx.In

	Logger              *zap.Logger
	DistanceControlRepo distancecontrolports.DistanceControlRepository
	DistanceProfileRepo distanceprofileports.DistanceProfileRepository
	IntegrationService  *integrationservice.Service
}

type resolveRuntime func(ctx context.Context, tenantInfo pagination.TenantInfo) tenantRuntime

type service struct {
	l        *zap.Logger
	resolve  resolveRuntime
	legs     *ttlcache.Cache[string, routeestimateports.LegEstimate]
	runtimes *ttlcache.Cache[string, tenantRuntime]
}

func New(p Params) routeestimateports.RouteEstimator {
	s := newService(p.Logger, nil)
	s.resolve = func(ctx context.Context, tenantInfo pagination.TenantInfo) tenantRuntime {
		control, err := p.DistanceControlRepo.Get(ctx, tenantInfo)
		if err != nil {
			control = nil
		}
		runtime := pcmilerruntime.Resolve(ctx, &pcmilerruntime.ResolveParams{
			Profiles:     p.DistanceProfileRepo,
			Integrations: p.IntegrationService,
			TenantInfo:   tenantInfo,
			Control:      control,
			Purpose:      distancecontrol.PurposeEtaOutOfRoute,
		})
		resolved := tenantRuntime{options: runtime.Options, ready: runtime.Ready}
		if runtime.Ready {
			resolved.client = runtime.Client
		}
		if runtime.Profile != nil {
			resolved.scope = runtime.Profile.ID.String() + ":" + strconv.FormatInt(
				runtime.Profile.UpdatedAt,
				10,
			)
		}
		return resolved
	}
	return s
}

func newService(logger *zap.Logger, resolve resolveRuntime) *service {
	return &service{
		l:       logger.Named("service.route-estimate"),
		resolve: resolve,
		legs: ttlcache.New[string, routeestimateports.LegEstimate](ttlcache.Options{
			TTL:        routedLegTTL,
			MaxEntries: routedLegCapacity,
		}),
		runtimes: ttlcache.New[string, tenantRuntime](ttlcache.Options{
			TTL:        runtimeTTL,
			MaxEntries: runtimeCapacity,
		}),
	}
}

func (s *service) EstimateLegs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	legs []routeestimateports.Leg,
) ([]routeestimateports.LegEstimate, error) {
	if len(legs) > routeestimateports.MaxLegsPerEstimate {
		return nil, fmt.Errorf(
			"estimate %d legs: at most %d per call",
			len(legs),
			routeestimateports.MaxLegsPerEstimate,
		)
	}

	out := make([]routeestimateports.LegEstimate, len(legs))
	for i, leg := range legs {
		out[i] = planned(leg)
	}
	if len(legs) == 0 {
		return out, nil
	}

	runtime := s.runtimeFor(ctx, tenantInfo)
	if !runtime.ready || runtime.client == nil {
		return out, nil
	}

	pending := make(map[string][]int, len(legs))
	routes := make([]pcmiler.RouteRequest, 0, len(legs))
	for i, leg := range legs {
		if out[i].Miles == 0 {
			out[i].Routed = true
			continue
		}
		key := legKey(runtime.scope, tenantInfo, leg)
		if cached, ok := s.legs.Get(key); ok {
			out[i] = cached
			continue
		}
		if _, queued := pending[key]; !queued {
			routes = append(routes, pcmiler.RouteRequest{
				RouteID: key,
				Stops:   []pcmiler.Stop{stopAt(leg.From), stopAt(leg.To)},
				Options: runtime.options,
			})
		}
		pending[key] = append(pending[key], i)
	}
	if len(routes) == 0 {
		return out, nil
	}

	results, err := runtime.client.Mileage(ctx, routes)
	if err != nil {
		s.l.Warn("routed drive time unavailable, using planning speed",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Int("legs", len(routes)),
			zap.Error(err))
		return out, nil
	}

	for _, result := range results {
		indexes, ok := pending[result.RouteID]
		if !ok || result.Distance <= 0 {
			continue
		}
		estimate := routeestimateports.LegEstimate{
			Miles:        result.Distance,
			DriveSeconds: result.DriveSeconds,
			Routed:       true,
		}
		if estimate.DriveSeconds <= 0 {
			estimate.DriveSeconds = geoutils.PlanningDriveSeconds(result.Distance)
		}
		s.legs.Set(result.RouteID, estimate)
		for _, index := range indexes {
			out[index] = estimate
		}
	}

	return out, nil
}

func (s *service) runtimeFor(ctx context.Context, tenantInfo pagination.TenantInfo) tenantRuntime {
	key := tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String()
	if cached, ok := s.runtimes.Get(key); ok {
		return cached
	}
	runtime := s.resolve(ctx, tenantInfo)
	s.runtimes.Set(key, runtime)
	return runtime
}

func planned(leg routeestimateports.Leg) routeestimateports.LegEstimate {
	miles := geoutils.PlanningRoadMiles(
		leg.From.Latitude,
		leg.From.Longitude,
		leg.To.Latitude,
		leg.To.Longitude,
	)
	return routeestimateports.LegEstimate{
		Miles:        miles,
		DriveSeconds: geoutils.PlanningDriveSeconds(miles),
	}
}

func stopAt(point routeestimateports.Point) pcmiler.Stop {
	latitude, longitude := point.Latitude, point.Longitude
	return pcmiler.Stop{Latitude: &latitude, Longitude: &longitude}
}

func legKey(scope string, tenantInfo pagination.TenantInfo, leg routeestimateports.Leg) string {
	return tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String() + ":" + scope + ":" +
		coordinate(leg.From.Latitude) + "," + coordinate(leg.From.Longitude) + ">" +
		coordinate(leg.To.Latitude) + "," + coordinate(leg.To.Longitude)
}

func coordinate(value float64) string {
	scale := math.Pow10(coordinatePlaces)
	return strconv.FormatFloat(math.Round(value*scale)/scale, 'f', coordinatePlaces, 64)
}
