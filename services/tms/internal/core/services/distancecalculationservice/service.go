package distancecalculationservice

import (
	"context"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/core/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const distancePrecision = 100

type Params struct {
	fx.In

	Logger               *zap.Logger
	ShipmentRepo         repositories.ShipmentRepository
	DistanceOverrideRepo repositories.DistanceOverrideRepository
	DistanceControlRepo  repositories.DistanceControlRepository
	DistanceProfileRepo  repositories.DistanceProfileRepository
	DistanceCalcRepo     repositories.DistanceCalculationRepository
	StoredMileageRepo    repositories.StoredMileageRepository
	StoredMileageBuffer  repositories.StoredMileageBufferRepository
	IntegrationService   *integrationservice.Service
}

type Service struct {
	l                    *zap.Logger
	shipmentRepo         repositories.ShipmentRepository
	distanceOverrideRepo repositories.DistanceOverrideRepository
	distanceControlRepo  repositories.DistanceControlRepository
	distanceProfileRepo  repositories.DistanceProfileRepository
	distanceCalcRepo     repositories.DistanceCalculationRepository
	storedMileageRepo    repositories.StoredMileageRepository
	storedMileageBuffer  repositories.StoredMileageBufferRepository
	integrationService   *integrationservice.Service
}

func New(p Params) services.DistanceCalculationService {
	return &Service{
		l:                    p.Logger.Named("service.distance-calculation"),
		shipmentRepo:         p.ShipmentRepo,
		distanceOverrideRepo: p.DistanceOverrideRepo,
		distanceControlRepo:  p.DistanceControlRepo,
		distanceProfileRepo:  p.DistanceProfileRepo,
		distanceCalcRepo:     p.DistanceCalcRepo,
		storedMileageRepo:    p.StoredMileageRepo,
		storedMileageBuffer:  p.StoredMileageBuffer,
		integrationService:   p.IntegrationService,
	}
}

func (s *Service) ResolveForShipment(
	ctx context.Context,
	entity *shipment.Shipment,
) (*services.DistanceCalculationResponse, error) {
	if entity == nil {
		return nil, errortypes.NewBusinessError("shipment is required")
	}

	resp := &services.DistanceCalculationResponse{
		ShipmentID: entity.ID,
		Moves:      make([]services.DistanceMoveResult, 0, len(entity.Moves)),
	}

	pcRuntimeByPurpose := make(map[string]pcmilerRuntime, 2)
	control, controlErr := s.distanceControlRepo.EnsureDefault(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if controlErr != nil {
		return nil, controlErr
	}
	hazmatTypes := hazmatTypesForShipment(entity)
	pcRequests := make([]pcmiler.RouteRequest, 0, len(entity.Moves))
	pcTargets := make(map[string]pcmilerMoveTarget, len(entity.Moves))

	for idx, move := range orderedMoves(entity.Moves) {
		if move == nil {
			continue
		}
		signature := buildRouteSignature(entity.CustomerID, move)
		now := timeutils.NowUnix()
		if !canResolveMoveDistance(move) {
			distance := applyManualDistance(move, signature, now)
			resp.Moves = append(resp.Moves, moveResult(move, idx, nil))
			resp.TotalDistance = addDistance(resp.TotalDistance, distance)
			continue
		}

		distance, ok, err := s.distanceOverride(ctx, entity, signature)
		if err != nil {
			return nil, err
		}
		if ok {
			applyMoveDistance(moveDistanceParams{
				move:         move,
				distance:     distance,
				source:       distancecalculation.SourceOverride,
				signature:    signature,
				calculatedAt: now,
			})
			resp.Moves = append(resp.Moves, moveResult(move, idx, nil))
			resp.TotalDistance = addDistance(resp.TotalDistance, distance)
			continue
		}

		runtime := s.runtimeForPurpose(ctx, entity, control, movePurpose(move), hazmatTypes, pcRuntimeByPurpose)
		if runtime.ready {
			storedDistance, storedOK, storedErr := s.storedMileage(
				ctx,
				entity,
				move,
				control,
				runtime.profile,
				runtime.options,
				hazmatTypes,
			)
			if storedErr != nil {
				return nil, storedErr
			}
			if storedOK {
				applyMoveDistance(moveDistanceParams{
					move:         move,
					distance:     storedDistance.Distance,
					source:       distancecalculation.SourceStoredMileage,
					provider:     storedDistance.Provider,
					signature:    storedDistance.RouteSignature,
					dataVersion:  storedDistance.DataVersion,
					routingType:  storedDistance.RoutingType,
					distanceUnit: runtime.options.DistanceUnits,
					profileID:    storedDistance.DistanceProfileID.String(),
					profileName:  storedDistance.DistanceProfileName,
					metadata: map[string]any{
						"storedMileageId":     storedDistance.ID.String(),
						"distanceProfileId":   storedDistance.DistanceProfileID.String(),
						"distanceProfileName": storedDistance.DistanceProfileName,
						"storedDistanceUnits": storedDistance.DistanceUnits,
					},
					calculatedAt: now,
				})
				resp.Moves = append(resp.Moves, moveResult(move, idx, nil))
				resp.TotalDistance = addDistance(resp.TotalDistance, storedDistance.Distance)
				s.incrementStoredMileageHit(entity, storedDistance.ID)
				continue
			}
			route, ok := buildPCMilerRoute(move, runtime.options, signature)
			if ok {
				pcTargets[route.RouteID] = pcmilerMoveTarget{
					move:    move,
					index:   idx,
					profile: runtime.profile,
					options: runtime.options,
				}
				pcRequests = append(pcRequests, route)
				continue
			}
		}

		distance = applyManualDistance(move, signature, now)
		resp.Moves = append(resp.Moves, moveResult(move, idx, nil))
		resp.TotalDistance = addDistance(resp.TotalDistance, distance)
	}

	if len(pcRequests) == 0 {
		return resp, nil
	}

	pcClient := firstReadyRuntime(pcRuntimeByPurpose).client
	pcResults, err := pcClient.Mileage(ctx, pcRequests)
	if err != nil {
		s.l.Warn("PC*Miler mileage failed, preserving manual distances", zap.Error(err))
		for _, target := range pcTargets {
			distance := applyManualDistance(
				target.move,
				target.move.DistanceRouteSignature,
				timeutils.NowUnix(),
			)
			resp.Moves = append(resp.Moves, moveResult(target.move, target.index, nil))
			resp.TotalDistance = addDistance(resp.TotalDistance, distance)
		}
		return resp, nil
	}

	resolvedRoutes := make(map[string]struct{}, len(pcResults))
	for _, result := range pcResults {
		target, ok := pcTargets[result.RouteID]
		if !ok || target.move == nil {
			continue
		}
		resolvedRoutes[result.RouteID] = struct{}{}
		applyMoveDistance(moveDistanceParams{
			move:         target.move,
			distance:     result.Distance,
			source:       distancecalculation.SourcePCMiler,
			provider:     string(integration.TypePCMiler),
			signature:    target.move.DistanceRouteSignature,
			dataVersion:  target.options.DataVersion,
			routingType:  target.options.RoutingType,
			distanceUnit: target.options.DistanceUnits,
			profileID:    target.profile.ID.String(),
			profileName:  target.profile.Name,
			metadata: map[string]any{
				"warnings":            result.Warnings,
				"distanceProfileId":   target.profile.ID.String(),
				"distanceProfileName": target.profile.Name,
			},
			calculatedAt: timeutils.NowUnix(),
		})
		resp.Moves = append(resp.Moves, moveResult(target.move, target.index, result.Warnings))
		resp.TotalDistance = addDistance(resp.TotalDistance, result.Distance)
		s.enqueueStoredMileageCandidate(
			ctx,
			entity,
			target.move,
			target.profile,
			target.options,
			result,
			control,
			hazmatTypes,
		)
	}
	for routeID, target := range pcTargets {
		if _, ok := resolvedRoutes[routeID]; ok {
			continue
		}
		distance := applyManualDistance(
			target.move,
			target.move.DistanceRouteSignature,
			timeutils.NowUnix(),
		)
		resp.Moves = append(resp.Moves, moveResult(target.move, target.index, nil))
		resp.TotalDistance = addDistance(resp.TotalDistance, distance)
	}

	sort.SliceStable(resp.Moves, func(i, j int) bool {
		return resp.Moves[i].MoveIndex < resp.Moves[j].MoveIndex
	})

	return resp, nil
}

type pcmilerMoveTarget struct {
	move    *shipment.ShipmentMove
	index   int
	profile *distanceprofile.DistanceProfile
	options pcmiler.RouteOptions
}

type pcmilerRuntime struct {
	client  *pcmiler.Client
	options pcmiler.RouteOptions
	profile *distanceprofile.DistanceProfile
	ready   bool
}

func (s *Service) RecalculateShipment(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*services.DistanceCalculationResponse, error) {
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	resp, err := s.ResolveForShipment(ctx, entity)
	if err != nil {
		return nil, err
	}
	if err = s.persistMoveDistances(ctx, entity.Moves); err != nil {
		return nil, err
	}
	s.logRuns(ctx, entity, resp)
	return resp, nil
}

func (s *Service) persistMoveDistances(ctx context.Context, moves []*shipment.ShipmentMove) error {
	for _, move := range moves {
		if move == nil || move.ID.IsNil() {
			continue
		}
		if err := s.distanceCalcRepo.UpdateMoveDistance(ctx, move); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) logRuns(
	ctx context.Context,
	entity *shipment.Shipment,
	resp *services.DistanceCalculationResponse,
) {
	if resp == nil {
		return
	}
	for _, result := range resp.Moves {
		run := &distancecalculation.Run{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			ShipmentID:     entity.ID,
			ShipmentMoveID: result.MoveID,
			Provider:       result.Provider,
			Source:         result.Source,
			RequestSummary: map[string]any{
				"moveIndex":             result.MoveIndex,
				"distance_profile_id":   result.DistanceProfileID,
				"distance_profile_name": result.DistanceProfileName,
			},
			ResponseSummary: map[string]any{
				"distance":              result.Distance,
				"routingType":           result.RoutingType,
				"dataVersion":           result.DataVersion,
				"distance_profile_id":   result.DistanceProfileID,
				"distance_profile_name": result.DistanceProfileName,
				"warnings":              result.Warnings,
			},
			Status: "Success",
		}
		if err := s.distanceCalcRepo.CreateRun(ctx, run); err != nil {
			s.l.Warn("failed to write distance calculation run", zap.Error(err))
		}
	}
}

func (s *Service) pcmilerRuntime(
	ctx context.Context,
	entity *shipment.Shipment,
	control *distancecontrol.DistanceControl,
	purpose string,
) (*pcmiler.Client, pcmiler.RouteOptions, *distanceprofile.DistanceProfile, bool) {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	var profile *distanceprofile.DistanceProfile
	var err error
	if control != nil {
		profileID := control.ProfileIDForPurpose(purpose)
		profile, err = s.distanceProfileRepo.GetByID(ctx, repositories.GetDistanceProfileByIDRequest{
			ID:         profileID,
			TenantInfo: tenantInfo,
		})
	} else {
		profile, err = s.distanceProfileRepo.EnsureDefault(ctx, tenantInfo)
	}
	if err != nil || profile.Status != distanceprofile.StatusActive {
		return nil, pcmiler.RouteOptions{}, nil, false
	}

	cfg, err := s.integrationService.GetRuntimeConfig(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}, integration.TypePCMiler)
	if err != nil || !cfg.Ready {
		return nil, pcmiler.RouteOptions{}, nil, false
	}

	client, err := pcmiler.New(pcmiler.Config{
		APIKey:  cfg.Config["apiKey"],
		BaseURL: cfg.Config["baseUrl"],
	})
	if err != nil {
		return nil, pcmiler.RouteOptions{}, nil, false
	}

	return client, profile.RouteOptions(), profile, true
}

func (s *Service) runtimeForPurpose(
	ctx context.Context,
	entity *shipment.Shipment,
	control *distancecontrol.DistanceControl,
	purpose string,
	hazmatTypes []string,
	cache map[string]pcmilerRuntime,
) pcmilerRuntime {
	if runtime, ok := cache[purpose]; ok {
		return runtime
	}
	client, options, profile, ready := s.pcmilerRuntime(ctx, entity, control, purpose)
	options.Hazmat = hazmatTypes
	runtime := pcmilerRuntime{
		client:  client,
		options: options,
		profile: profile,
		ready:   ready,
	}
	cache[purpose] = runtime
	return runtime
}

func firstReadyRuntime(cache map[string]pcmilerRuntime) pcmilerRuntime {
	for _, runtime := range cache {
		if runtime.ready {
			return runtime
		}
	}
	return pcmilerRuntime{}
}

func (s *Service) distanceOverride(
	ctx context.Context,
	entity *shipment.Shipment,
	signature string,
) (float64, bool, error) {
	tenantInfo := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	exact, err := s.distanceOverrideRepo.GetByRouteSignature(ctx, tenantInfo, signature)
	if err == nil {
		return exact.Distance, true, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return 0, false, err
	}

	wildcard := wildcardRouteSignature(signature)
	if wildcard == signature {
		return 0, false, nil
	}
	override, err := s.distanceOverrideRepo.GetByRouteSignature(ctx, tenantInfo, wildcard)
	if err == nil {
		return override.Distance, true, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return 0, false, err
	}

	return 0, false, nil
}

func (s *Service) storedMileage(
	ctx context.Context,
	entity *shipment.Shipment,
	move *shipment.ShipmentMove,
	control *distancecontrol.DistanceControl,
	profile *distanceprofile.DistanceProfile,
	options pcmiler.RouteOptions,
	hazmatTypes []string,
) (*storedmileage.StoredMileage, bool, error) {
	if control == nil || profile == nil || !control.StoreMileage {
		return nil, false, nil
	}
	candidate, ok := buildStoredMileageCandidate(
		entity,
		move,
		profile,
		options,
		control.PostalCodeFallbackToCity,
		0,
		nil,
		hazmatTypes,
	)
	if !ok {
		return nil, false, nil
	}
	found, err := s.storedMileageRepo.Lookup(ctx, repositories.StoredMileageLookupRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		RouteHash:         candidate.RouteHash,
		DistanceUnits:     candidate.DistanceUnits,
		RoutingType:       candidate.RoutingType,
		Method:            candidate.Method,
		DistanceProfileID: candidate.DistanceProfileID,
		HazmatSignature:   candidate.HazmatSignature,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	found.Distance = roundDistance(
		storedmileage.ConvertDistance(found.Distance, found.DistanceUnits, options.DistanceUnits),
	)
	return found, true, nil
}

func (s *Service) incrementStoredMileageHit(entity *shipment.Shipment, storedMileageID pulid.ID) {
	if storedMileageID.IsNil() {
		return
	}
	go func() {
		ctx := context.WithoutCancel(context.Background())
		if err := s.storedMileageRepo.IncrementHit(ctx, storedMileageID, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		}); err != nil {
			s.l.Warn("failed to increment stored mileage hit", zap.Error(err))
		}
	}()
}

func (s *Service) enqueueStoredMileageCandidate(
	ctx context.Context,
	entity *shipment.Shipment,
	move *shipment.ShipmentMove,
	profile *distanceprofile.DistanceProfile,
	options pcmiler.RouteOptions,
	result pcmiler.RouteMileage,
	control *distancecontrol.DistanceControl,
	hazmatTypes []string,
) {
	if control == nil || !control.AutoCreateStoredMileage || s.storedMileageBuffer == nil {
		return
	}
	candidate, ok := buildStoredMileageCandidate(
		entity,
		move,
		profile,
		options,
		control.PostalCodeFallbackToCity,
		result.Distance,
		map[string]any{"warnings": result.Warnings, "rawSummary": result.RawSummary},
		hazmatTypes,
	)
	if !ok {
		return
	}
	if err := s.storedMileageBuffer.Push(ctx, candidate); err != nil {
		s.l.Warn("failed to buffer stored mileage candidate", zap.Error(err))
	}
}

func buildStoredMileageCandidate(
	entity *shipment.Shipment,
	move *shipment.ShipmentMove,
	profile *distanceprofile.DistanceProfile,
	options pcmiler.RouteOptions,
	postalCodeFallbackToCity bool,
	distance float64,
	metadata map[string]any,
	hazmatTypes []string,
) (*storedmileage.StoredMileage, bool) {
	if entity == nil || move == nil || profile == nil {
		return nil, false
	}
	stops := orderedStops(move.Stops)
	if len(stops) < 2 {
		return nil, false
	}
	keys := make([]storedmileage.StopKey, 0, len(stops))
	for _, stop := range stops {
		if stop == nil || stop.Location == nil {
			return nil, false
		}
		key, ok := storedMileageStopKey(stop.Location, options, postalCodeFallbackToCity)
		if !ok {
			return nil, false
		}
		keys = append(keys, key)
	}
	routeSignature := storedMileageRouteSignature(keys, options, profile, hazmatTypes)
	candidate := &storedmileage.StoredMileage{
		OrganizationID:      entity.OrganizationID,
		BusinessUnitID:      entity.BusinessUnitID,
		Status:              storedmileage.StatusActive,
		OriginKey:           keys[0],
		DestinationKey:      keys[len(keys)-1],
		IntermediateKeys:    keys[1 : len(keys)-1],
		RouteSignature:      routeSignature,
		RouteHash:           hashutils.SHA256Hex(routeSignature),
		Distance:            roundDistance(distance),
		DistanceUnits:       options.DistanceUnits,
		Provider:            string(integration.TypePCMiler),
		Source:              storedmileage.SourcePCMiler,
		RoutingType:         options.RoutingType,
		Method:              optionsGranularity(options),
		LocationGranularity: optionsGranularity(options),
		DataVersion:         options.DataVersion,
		DistanceProfileID:   profile.ID,
		DistanceProfileName: profile.Name,
		HazmatTypes:         hazmatTypes,
		ProviderMetadata:    metadata,
	}
	candidate.ApplyDefaults()
	return candidate, true
}

func storedMileageStopKey(
	loc *location.Location,
	options pcmiler.RouteOptions,
	postalCodeFallbackToCity bool,
) (storedmileage.StopKey, bool) {
	state := ""
	if loc.State != nil {
		state = loc.State.Abbreviation
	}
	method := optionsGranularity(options)
	key := storedmileage.StopKey{
		Method:     method,
		City:       normalizeKeyPart(loc.City),
		State:      normalizeKeyPart(state),
		PostalCode: normalizeKeyPart(loc.PostalCode),
		PlaceID:    strings.TrimSpace(loc.PlaceID),
	}
	switch method {
	case "Coordinates":
		if loc.Latitude == nil || loc.Longitude == nil {
			return storedmileage.StopKey{}, false
		}
		key.Coordinates = []float64{*loc.Latitude, *loc.Longitude}
		key.Key = strings.Join([]string{
			method,
			strconv.FormatFloat(*loc.Latitude, 'f', 6, 64),
			strconv.FormatFloat(*loc.Longitude, 'f', 6, 64),
		}, "|")
	case "TrimblePlaceId":
		if strings.TrimSpace(loc.PlaceID) == "" {
			return storedmileage.StopKey{}, false
		}
		key.Key = method + "|" + strings.TrimSpace(loc.PlaceID)
	case "PostalCode":
		if key.PostalCode == "" {
			if !postalCodeFallbackToCity {
				return storedmileage.StopKey{}, false
			}
			if key.City == "" || key.State == "" {
				return storedmileage.StopKey{}, false
			}
			key.Method = "CityState"
			key.Key = "CityState|" + key.City + "|" + key.State
			return key, true
		}
		key.Key = method + "|" + key.PostalCode
	default:
		if key.City == "" || key.State == "" || strings.TrimSpace(loc.AddressLine1) == "" {
			return storedmileage.StopKey{}, false
		}
		key.Key = method + "|" + normalizeKeyPart(loc.AddressLine1) + "|" + key.City + "|" + key.State + "|" + key.PostalCode
	}
	return key, true
}

func storedMileageRouteSignature(
	keys []storedmileage.StopKey,
	options pcmiler.RouteOptions,
	profile *distanceprofile.DistanceProfile,
	hazmatTypes []string,
) string {
	parts := make([]string, 0, len(keys)+6)
	parts = append(parts, options.DistanceUnits, options.RoutingType, optionsGranularity(options), profile.ID.String())
	parts = append(parts, storedmileage.HazmatSignature(hazmatTypes))
	for _, key := range keys {
		parts = append(parts, key.Key)
	}
	return strings.Join(parts, "|")
}

func normalizeKeyPart(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func buildPCMilerRoute(
	move *shipment.ShipmentMove,
	options pcmiler.RouteOptions,
	signature string,
) (pcmiler.RouteRequest, bool) {
	stops := orderedStops(move.Stops)
	if len(stops) < 2 {
		return pcmiler.RouteRequest{}, false
	}

	pcStops := make([]pcmiler.Stop, 0, len(stops))
	for _, stop := range stops {
		if stop == nil || stop.Location == nil {
			return pcmiler.RouteRequest{}, false
		}
		pcStops = append(pcStops, locationToPCMilerStop(stop.Location, options))
	}

	move.DistanceRouteSignature = signature
	return pcmiler.RouteRequest{
		RouteID: signature,
		Stops:   pcStops,
		Options: options,
	}, true
}

func locationToPCMilerStop(loc *location.Location, options pcmiler.RouteOptions) pcmiler.Stop {
	state := ""
	if loc.State != nil {
		state = loc.State.Abbreviation
	}

	stop := pcmiler.Stop{City: loc.City, State: state, PostalCode: loc.PostalCode}
	switch optionsGranularity(options) {
	case "StreetAddress":
		stop.AddressLine = loc.AddressLine1
	case "Coordinates":
		stop.Latitude = loc.Latitude
		stop.Longitude = loc.Longitude
	case "TrimblePlaceId":
		stop.TrimblePlaceID = loc.PlaceID
	}
	return stop
}

func optionsGranularity(options pcmiler.RouteOptions) string {
	granularity := stringutils.WithDefault(options.LocationGranularity, "PostalCode")
	switch {
	case strings.EqualFold(granularity, "StreetAddress"):
		return "StreetAddress"
	case strings.EqualFold(granularity, "Coordinates"):
		return "Coordinates"
	case strings.EqualFold(granularity, "TrimblePlaceId"):
		return "TrimblePlaceId"
	default:
		return "PostalCode"
	}
}

type moveDistanceParams struct {
	move         *shipment.ShipmentMove
	distance     float64
	source       string
	provider     string
	signature    string
	dataVersion  string
	routingType  string
	distanceUnit string
	profileID    string
	profileName  string
	metadata     map[string]any
	calculatedAt int64
}

func applyMoveDistance(params moveDistanceParams) {
	params.distance = roundDistance(params.distance)
	params.move.Distance = &params.distance
	params.move.DistanceSource = params.source
	params.move.DistanceProvider = params.provider
	params.move.DistanceRouteSignature = params.signature
	params.move.DistanceDataVersion = params.dataVersion
	params.move.DistanceRoutingType = params.routingType
	params.move.DistanceUnits = params.distanceUnit
	params.move.DistanceCalculatedAt = &params.calculatedAt
	params.move.DistanceMetadata = params.metadata
}

func applyManualDistance(move *shipment.ShipmentMove, signature string, calculatedAt int64) float64 {
	distance := roundDistance(manualDistance(move))
	applyMoveDistance(moveDistanceParams{
		move:         move,
		distance:     distance,
		source:       distancecalculation.SourceManual,
		signature:    signature,
		calculatedAt: calculatedAt,
	})

	return distance
}

func moveResult(move *shipment.ShipmentMove, idx int, warnings []string) services.DistanceMoveResult {
	distance := roundDistance(manualDistance(move))
	if move.Distance != nil {
		distance = roundDistance(*move.Distance)
	}
	calculatedAt := int64(0)
	if move.DistanceCalculatedAt != nil {
		calculatedAt = *move.DistanceCalculatedAt
	}
	return services.DistanceMoveResult{
		MoveID:              move.ID,
		MoveIndex:           idx,
		Distance:            distance,
		Source:              move.DistanceSource,
		Provider:            move.DistanceProvider,
		RoutingType:         move.DistanceRoutingType,
		DataVersion:         move.DistanceDataVersion,
		DistanceUnits:       move.DistanceUnits,
		DistanceProfileID:   profileIDFromMetadata(move.DistanceMetadata),
		DistanceProfileName: profileNameFromMetadata(move.DistanceMetadata),
		Warnings:            warnings,
		CalculatedAt:        calculatedAt,
	}
}

func addDistance(total, distance float64) float64 {
	return roundDistance(total + distance)
}

func roundDistance(distance float64) float64 {
	if math.IsNaN(distance) || math.IsInf(distance, 0) {
		return distance
	}
	return math.Round(distance*distancePrecision) / distancePrecision
}

func profileIDFromMetadata(metadata map[string]any) string {
	if value, ok := metadata["distanceProfileId"].(string); ok {
		return value
	}
	return ""
}

func profileNameFromMetadata(metadata map[string]any) string {
	if value, ok := metadata["distanceProfileName"].(string); ok {
		return value
	}
	return ""
}

func buildRouteSignature(customerID pulid.ID, move *shipment.ShipmentMove) string {
	parts := make([]string, 0, len(move.Stops))
	for _, stop := range orderedStops(move.Stops) {
		if stop != nil && !stop.LocationID.IsNil() {
			parts = append(parts, stop.LocationID.String())
		}
	}

	scope := "*"
	if !customerID.IsNil() {
		scope = customerID.String()
	}
	return scope + "|" + strings.Join(parts, ">")
}

func wildcardRouteSignature(signature string) string {
	_, route, ok := strings.Cut(signature, "|")
	if !ok {
		return signature
	}
	return "*|" + route
}

func orderedMoves(moves []*shipment.ShipmentMove) []*shipment.ShipmentMove {
	ordered := append([]*shipment.ShipmentMove(nil), moves...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Sequence < ordered[j].Sequence
	})
	return ordered
}

func orderedStops(stops []*shipment.Stop) []*shipment.Stop {
	ordered := append([]*shipment.Stop(nil), stops...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Sequence < ordered[j].Sequence
	})
	return ordered
}

func canResolveMoveDistance(move *shipment.ShipmentMove) bool {
	if move == nil {
		return false
	}

	var locationCount int
	for _, stop := range orderedStops(move.Stops) {
		if stop != nil && !stop.LocationID.IsNil() {
			locationCount++
		}
	}

	return locationCount >= 2
}

func movePurpose(move *shipment.ShipmentMove) string {
	if move != nil && !move.Loaded {
		return distancecontrol.PurposeEmptyMove
	}
	return distancecontrol.PurposeLoadedMove
}

func manualDistance(move *shipment.ShipmentMove) float64 {
	if move == nil || move.Distance == nil {
		return 0
	}
	return *move.Distance
}

func hazmatTypesForShipment(entity *shipment.Shipment) []string {
	if entity == nil || len(entity.Commodities) == 0 {
		return []string{}
	}

	values := make(map[string]struct{}, len(entity.Commodities))
	for _, item := range entity.Commodities {
		if item == nil || item.Commodity == nil || item.Commodity.HazardousMaterial == nil {
			continue
		}
		for _, value := range hazmatTypesForMaterial(item.Commodity.HazardousMaterial) {
			values[value] = struct{}{}
		}
	}

	types := make([]string, 0, len(values))
	for value := range values {
		types = append(types, value)
	}
	sort.Strings(types)
	return types
}

func hazmatTypesForMaterial(material *hazardousmaterial.HazardousMaterial) []string {
	if material == nil {
		return []string{}
	}

	values := make([]string, 0, 2)
	switch material.Class {
	case hazardousmaterial.HazardousClass1,
		hazardousmaterial.HazardousClass1And1,
		hazardousmaterial.HazardousClass1And2,
		hazardousmaterial.HazardousClass1And3,
		hazardousmaterial.HazardousClass1And4,
		hazardousmaterial.HazardousClass1And5,
		hazardousmaterial.HazardousClass1And6:
		values = append(values, "Explosives")
	case hazardousmaterial.HazardousClass2And3:
		values = append(values, "Inhalants")
	case hazardousmaterial.HazardousClass2And1,
		hazardousmaterial.HazardousClass3,
		hazardousmaterial.HazardousClass4And1,
		hazardousmaterial.HazardousClass4And2,
		hazardousmaterial.HazardousClass4And3:
		values = append(values, "Flammable")
	case hazardousmaterial.HazardousClass7:
		values = append(values, "Radioactive")
	case hazardousmaterial.HazardousClass8:
		values = append(values, "Caustic")
	default:
		values = append(values, "General")
	}

	if material.InhalationHazard && !slices.Contains(values, "Inhalants") {
		values = append(values, "Inhalants")
	}
	if material.MarinePollutant {
		values = append(values, "HarmfulToWater")
	}

	return values
}
