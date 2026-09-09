package distancecalculationservice

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

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
	"github.com/emoss08/trenova/internal/core/temporaljobs/iftajobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/countryutils"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	distancePrecision = 100

	jurisdictionAbsoluteTolerance = 0.5
	jurisdictionRelativeTolerance = 0.005
	jurisdictionRunPurpose        = "JurisdictionMiles"
	runStatusSuccess              = "Success"
	runStatusFailed               = "Failed"

	JurisdictionAttributionAttributed   = "Attributed"
	JurisdictionAttributionUnattributed = "Unattributed"
	JurisdictionAttributionMismatch     = "Mismatch"
)

type mileageClient interface {
	Mileage(ctx context.Context, routes []pcmiler.RouteRequest) ([]pcmiler.RouteMileage, error)
}

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
	JurisdictionMileRepo repositories.ShipmentMoveJurisdictionMileRepository
	ShipmentMoveRepo     repositories.ShipmentMoveRepository
	WorkflowStarter      services.WorkflowStarter
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
	jurisdictionMileRepo repositories.ShipmentMoveJurisdictionMileRepository
	shipmentMoveRepo     repositories.ShipmentMoveRepository
	workflowStarter      services.WorkflowStarter
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
		jurisdictionMileRepo: p.JurisdictionMileRepo,
		shipmentMoveRepo:     p.ShipmentMoveRepo,
		workflowStarter:      p.WorkflowStarter,
		integrationService:   p.IntegrationService,
	}
}

func (s *Service) ResolveForShipment(
	ctx context.Context,
	entity *shipment.Shipment,
) (*services.DistanceCalculationResponse, error) {
	return s.resolveForShipment(ctx, entity, make(map[string]pcmilerRuntime, 2))
}

func (s *Service) resolveForShipment(
	ctx context.Context,
	entity *shipment.Shipment,
	pcRuntimeByPurpose map[string]pcmilerRuntime,
) (*services.DistanceCalculationResponse, error) {
	if entity == nil {
		return nil, errortypes.NewBusinessError("shipment is required")
	}

	resp := &services.DistanceCalculationResponse{
		ShipmentID: entity.ID,
		Moves:      make([]services.DistanceMoveResult, 0, len(entity.Moves)),
	}

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

		runtime := s.runtimeForPurpose(
			ctx,
			entity,
			control,
			movePurpose(move),
			hazmatTypes,
			pcRuntimeByPurpose,
		)
		if runtime.profile != nil {
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
			if storedOK && useStoredMileage(storedDistance, runtime) {
				warnings := applyMoveDistance(moveDistanceParams{
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
					calculatedAt:  now,
					jurisdictions: pcmilerJurisdictionsFromStored(storedDistance.JurisdictionDistances),
				})
				resp.Moves = append(resp.Moves, moveResult(move, idx, warnings))
				resp.TotalDistance = addDistance(resp.TotalDistance, storedDistance.Distance)
				s.incrementStoredMileageHit(entity, storedDistance.ID)
				continue
			}
			if runtime.ready {
				route, routeOK := buildPCMilerRoute(move, runtime.options, signature)
				if routeOK {
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
		warnings := applyMoveDistance(moveDistanceParams{
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
			calculatedAt:  timeutils.NowUnix(),
			jurisdictions: result.JurisdictionDistances,
		})
		resp.Moves = append(
			resp.Moves,
			moveResult(target.move, target.index, mergeWarnings(result.Warnings, warnings)),
		)
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
	client  mileageClient
	options pcmiler.RouteOptions
	profile *distanceprofile.DistanceProfile
	ready   bool
}

func useStoredMileage(stored *storedmileage.StoredMileage, runtime pcmilerRuntime) bool {
	if stored == nil {
		return false
	}
	if stored.HasJurisdictionBreakdown() {
		return true
	}
	return !(runtime.ready && runtime.options.StateReport)
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

func (s *Service) RecalculateMoveJurisdictionMiles(
	ctx context.Context,
	req services.RecalculateMoveJurisdictionMilesRequest,
) ([]*shipment.ShipmentMoveJurisdictionMile, error) {
	if req.ShipmentMoveID.IsNil() {
		return nil, errortypes.NewBusinessError("Shipment move is required")
	}
	move, err := s.shipmentMoveRepo.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            req.ShipmentMoveID,
		TenantInfo:        req.TenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}
	if !canResolveMoveDistance(move) {
		return nil, errortypes.NewBusinessError(
			"Jurisdiction miles need a move with at least two located stops",
		).WithParam("moveId", move.ID.String())
	}
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         move.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}
	control, err := s.distanceControlRepo.EnsureDefault(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	hazmatTypes := hazmatTypesForShipment(entity)
	runtime := s.runtimeForPurpose(
		ctx,
		entity,
		control,
		movePurpose(move),
		hazmatTypes,
		make(map[string]pcmilerRuntime, 1),
	)
	if !runtime.ready {
		return nil, errortypes.NewBusinessError(
			"PC*Miler is not ready; configure the integration before requesting jurisdiction miles",
		)
	}
	runtime.options.StateReport = true
	route, ok := buildPCMilerRoute(
		move,
		runtime.options,
		buildRouteSignature(entity.CustomerID, move),
	)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"Every stop on the move needs a location before jurisdiction miles can be requested",
		).WithParam("moveId", move.ID.String())
	}

	started := time.Now()
	results, err := runtime.client.Mileage(ctx, []pcmiler.RouteRequest{route})
	latency := time.Since(started).Milliseconds()
	if err != nil {
		s.logJurisdictionRun(ctx, jurisdictionRunParams{
			move:    move,
			profile: runtime.profile,
			latency: latency,
			err:     err,
		})
		return nil, fmt.Errorf("request jurisdiction miles for move %s: %w", move.ID, err)
	}
	result, found := findRouteResult(results, route.RouteID)
	if !found {
		err = errortypes.NewBusinessError("PC*Miler returned no route for the move").
			WithParam("moveId", move.ID.String())
		s.logJurisdictionRun(ctx, jurisdictionRunParams{
			move:    move,
			profile: runtime.profile,
			latency: latency,
			err:     err,
		})
		return nil, err
	}

	rows := buildJurisdictionMiles(moveDistanceParams{
		move:          move,
		distance:      result.Distance,
		source:        distancecalculation.SourcePCMiler,
		provider:      string(integration.TypePCMiler),
		dataVersion:   runtime.options.DataVersion,
		distanceUnit:  runtime.options.DistanceUnits,
		profileID:     runtime.profile.ID.String(),
		calculatedAt:  timeutils.NowUnix(),
		jurisdictions: result.JurisdictionDistances,
	})
	move.JurisdictionMiles = rows
	move.JurisdictionMilesDirty = true
	if err = s.jurisdictionMileRepo.ReplaceForMove(ctx, move); err != nil {
		return nil, err
	}
	s.enqueueStoredMileageCandidate(
		ctx,
		entity,
		move,
		runtime.profile,
		runtime.options,
		result,
		control,
		hazmatTypes,
	)
	s.logJurisdictionRun(ctx, jurisdictionRunParams{
		move:    move,
		profile: runtime.profile,
		result:  &result,
		rows:    rows,
		latency: latency,
	})
	return rows, nil
}

func (s *Service) BackfillJurisdictionMiles(
	ctx context.Context,
	req services.BackfillJurisdictionMilesRequest,
) (*services.BackfillJurisdictionMilesResult, error) {
	if req.Start <= 0 || req.End <= req.Start {
		return nil, errortypes.NewBusinessError("Period start must be before period end")
	}
	page, err := s.jurisdictionMileRepo.ListUnattributedMoves(
		ctx,
		repositories.UnattributedMovesRequest{
			TenantInfo: req.TenantInfo,
			Start:      req.Start,
			End:        req.End,
			Limit:      1,
		},
	)
	if err != nil {
		return nil, err
	}
	result := &services.BackfillJurisdictionMilesResult{
		DryRun:            req.DryRun,
		UnattributedMoves: page.TotalMoves,
		UnattributedMiles: page.TotalMiles,
	}
	if req.DryRun || page.TotalMoves == 0 {
		return result, nil
	}
	if !s.workflowStarter.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Background jobs are unavailable, so the jurisdiction backfill cannot be started",
		)
	}

	maxMoves := req.MaxMoves
	if maxMoves <= 0 {
		maxMoves = iftajobs.DefaultBackfillMaxMoves
	}
	tenantInfo := req.TenantInfo
	if !req.UserID.IsNil() {
		tenantInfo.UserID = req.UserID
	}
	workflowID := fmt.Sprintf(
		"ifta-jurisdiction-backfill-%s-%d-%d",
		req.TenantInfo.OrgID.String(),
		req.Start,
		req.End,
	)
	run, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       workflowID,
			TaskQueue:                                temporaltype.TaskQueueSystem.String(),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary: fmt.Sprintf(
				"Backfilling jurisdiction miles for up to %d of %d unattributed moves",
				maxMoves,
				page.TotalMoves,
			),
		},
		iftajobs.BackfillJurisdictionMilesWorkflowName,
		iftajobs.BackfillInput{
			TenantInfo: tenantInfo,
			Start:      req.Start,
			End:        req.End,
			MaxMoves:   maxMoves,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("start jurisdiction backfill workflow: %w", err)
	}
	result.Started = true
	result.WorkflowID = run.GetID()
	return result, nil
}

type jurisdictionRunParams struct {
	move    *shipment.ShipmentMove
	profile *distanceprofile.DistanceProfile
	result  *pcmiler.RouteMileage
	rows    []*shipment.ShipmentMoveJurisdictionMile
	latency int64
	err     error
}

func (s *Service) logJurisdictionRun(ctx context.Context, params jurisdictionRunParams) {
	move := params.move
	request := map[string]any{
		"purpose":     jurisdictionRunPurpose,
		"stateReport": true,
	}
	if params.profile != nil {
		request["distance_profile_id"] = params.profile.ID.String()
		request["distance_profile_name"] = params.profile.Name
	}
	run := &distancecalculation.Run{
		OrganizationID: move.OrganizationID,
		BusinessUnitID: move.BusinessUnitID,
		ShipmentID:     move.ShipmentID,
		ShipmentMoveID: move.ID,
		Provider:       string(integration.TypePCMiler),
		Source:         distancecalculation.SourcePCMiler,
		RequestSummary: request,
		Status:         runStatusSuccess,
		LatencyMillis:  params.latency,
	}
	if params.err != nil {
		run.Status = runStatusFailed
		run.ErrorMessage = params.err.Error()
		run.ResponseSummary = map[string]any{
			"jurisdictionCount":       0,
			"jurisdictionSum":         0.0,
			"jurisdictionAttribution": JurisdictionAttributionUnattributed,
		}
	} else if params.result != nil {
		comparison := params.result.Distance
		if move.Distance != nil {
			comparison = *move.Distance
		}
		attribution, sum := jurisdictionAttribution(comparison, move.DistanceUnits, params.rows)
		run.ResponseSummary = map[string]any{
			"distance":                params.result.Distance,
			"moveDistance":            manualDistance(move),
			"dataVersion":             params.result.DataVersion,
			"warnings":                params.result.Warnings,
			"jurisdictionCount":       len(params.rows),
			"jurisdictionSum":         sum,
			"jurisdictionAttribution": attribution,
		}
	}
	if err := s.distanceCalcRepo.CreateRun(ctx, run); err != nil {
		s.l.Warn("failed to write jurisdiction miles run", zap.Error(err))
	}
}

func findRouteResult(results []pcmiler.RouteMileage, routeID string) (pcmiler.RouteMileage, bool) {
	for _, result := range results {
		if result.RouteID == routeID {
			return result, true
		}
	}
	if len(results) == 1 && routeID == "" {
		return results[0], true
	}
	return pcmiler.RouteMileage{}, false
}

func (s *Service) persistMoveDistances(ctx context.Context, moves []*shipment.ShipmentMove) error {
	for _, move := range moves {
		if move == nil || move.ID.IsNil() {
			continue
		}
		if err := s.distanceCalcRepo.UpdateMoveDistance(ctx, move); err != nil {
			return err
		}
		if !move.JurisdictionMilesDirty {
			continue
		}
		if err := s.jurisdictionMileRepo.ReplaceForMove(ctx, move); err != nil {
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
	movesByID := make(map[pulid.ID]*shipment.ShipmentMove, len(entity.Moves))
	for _, move := range entity.Moves {
		if move != nil && !move.ID.IsNil() {
			movesByID[move.ID] = move
		}
	}
	for _, result := range resp.Moves {
		var rows []*shipment.ShipmentMoveJurisdictionMile
		if move := movesByID[result.MoveID]; move != nil {
			rows = move.JurisdictionMiles
		}
		attribution, jurisdictionSum := jurisdictionAttribution(
			result.Distance,
			result.DistanceUnits,
			rows,
		)
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
				"distance":                result.Distance,
				"routingType":             result.RoutingType,
				"dataVersion":             result.DataVersion,
				"distance_profile_id":     result.DistanceProfileID,
				"distance_profile_name":   result.DistanceProfileName,
				"warnings":                result.Warnings,
				"jurisdictionCount":       len(rows),
				"jurisdictionSum":         jurisdictionSum,
				"jurisdictionAttribution": attribution,
			},
			Status: runStatusSuccess,
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
		profile, err = s.distanceProfileRepo.GetByID(
			ctx,
			repositories.GetDistanceProfileByIDRequest{
				ID:         profileID,
				TenantInfo: tenantInfo,
			},
		)
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
		return nil, profile.RouteOptions(), profile, false
	}

	pcClient, err := pcmiler.New(pcmiler.Config{
		APIKey:  cfg.Config["apiKey"],
		BaseURL: cfg.Config["baseUrl"],
	})
	if err != nil {
		return nil, profile.RouteOptions(), profile, false
	}

	return pcClient, profile.RouteOptions(), profile, true
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
	pcClient, options, profile, ready := s.pcmilerRuntime(ctx, entity, control, purpose)
	options.Hazmat = hazmatTypes
	if purposeCapturesJurisdictions(purpose) {
		options.StateReport = control != nil && control.CaptureJurisdictionMiles
	}
	runtime := pcmilerRuntime{
		options: options,
		profile: profile,
		ready:   ready,
	}
	if ready {
		runtime.client = pcClient
	}
	cache[purpose] = runtime
	return runtime
}

func purposeCapturesJurisdictions(purpose string) bool {
	return purpose == distancecontrol.PurposeLoadedMove ||
		purpose == distancecontrol.PurposeEmptyMove
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
	if found.HasJurisdictionBreakdown() {
		found.JurisdictionDistances = convertStoredJurisdictions(
			found.JurisdictionDistances,
			found.DistanceUnits,
			options.DistanceUnits,
		)
	}
	return found, true, nil
}

func convertStoredJurisdictions(
	items []storedmileage.JurisdictionDistance,
	fromUnits, toUnits string,
) []storedmileage.JurisdictionDistance {
	converted := make([]storedmileage.JurisdictionDistance, 0, len(items))
	for _, item := range items {
		converted = append(converted, storedmileage.JurisdictionDistance{
			Country:  item.Country,
			Code:     item.Code,
			Distance: roundDistance(storedmileage.ConvertDistance(item.Distance, fromUnits, toUnits)),
			Toll:     roundDistance(storedmileage.ConvertDistance(item.Toll, fromUnits, toUnits)),
			Ferry:    roundDistance(storedmileage.ConvertDistance(item.Ferry, fromUnits, toUnits)),
		})
	}
	return converted
}

func pcmilerJurisdictionsFromStored(
	items []storedmileage.JurisdictionDistance,
) []pcmiler.JurisdictionDistance {
	converted := make([]pcmiler.JurisdictionDistance, 0, len(items))
	for _, item := range items {
		converted = append(converted, pcmiler.JurisdictionDistance(item))
	}
	return converted
}

func storedJurisdictionsFromPCMiler(
	items []pcmiler.JurisdictionDistance,
) []storedmileage.JurisdictionDistance {
	converted := make([]storedmileage.JurisdictionDistance, 0, len(items))
	for _, item := range items {
		converted = append(converted, storedmileage.JurisdictionDistance(item))
	}
	return converted
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
	candidate.JurisdictionDistances = storedJurisdictionsFromPCMiler(result.JurisdictionDistances)
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
		key.Key = method + "|" + normalizeKeyPart(
			loc.AddressLine1,
		) + "|" + key.City + "|" + key.State + "|" + key.PostalCode
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
	parts = append(
		parts,
		options.DistanceUnits,
		options.RoutingType,
		optionsGranularity(options),
		profile.ID.String(),
		storedmileage.HazmatSignature(hazmatTypes),
	)
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
	country := countryutils.ISO2UnitedStates
	if loc.State != nil {
		state = loc.State.Abbreviation
		country = countryutils.ISO3ToISO2OrDefault(
			loc.State.CountryIso3,
			countryutils.ISO2UnitedStates,
		)
	}

	stop := pcmiler.Stop{
		City:       loc.City,
		State:      state,
		PostalCode: loc.PostalCode,
		Country:    country,
	}
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
	move          *shipment.ShipmentMove
	distance      float64
	source        string
	provider      string
	signature     string
	dataVersion   string
	routingType   string
	distanceUnit  string
	profileID     string
	profileName   string
	metadata      map[string]any
	calculatedAt  int64
	jurisdictions []pcmiler.JurisdictionDistance
}

func applyMoveDistance(params moveDistanceParams) []string {
	params.distance = roundDistance(params.distance)
	rows := buildJurisdictionMiles(params)
	var warnings []string
	if warning, mismatch := jurisdictionMismatchWarning(
		params.distance,
		params.distanceUnit,
		rows,
	); mismatch {
		warnings = []string{warning}
		params.metadata = appendMetadataWarning(params.metadata, warning)
	}
	params.move.Distance = &params.distance
	params.move.DistanceSource = params.source
	params.move.DistanceProvider = params.provider
	params.move.DistanceRouteSignature = params.signature
	params.move.DistanceDataVersion = params.dataVersion
	params.move.DistanceRoutingType = params.routingType
	params.move.DistanceUnits = params.distanceUnit
	params.move.DistanceCalculatedAt = &params.calculatedAt
	params.move.DistanceMetadata = params.metadata
	params.move.JurisdictionMiles = rows
	params.move.JurisdictionMilesDirty = true
	return warnings
}

func buildJurisdictionMiles(params moveDistanceParams) []*shipment.ShipmentMoveJurisdictionMile {
	rows := make([]*shipment.ShipmentMoveJurisdictionMile, 0, len(params.jurisdictions))
	if params.move == nil ||
		params.source == distancecalculation.SourceManual ||
		params.source == distancecalculation.SourceOverride {
		return rows
	}
	units := normalizeJurisdictionUnits(params.distanceUnit)
	for _, item := range params.jurisdictions {
		code := strings.ToUpper(strings.TrimSpace(item.Code))
		if code == "" || item.Distance < 0 {
			continue
		}
		row := &shipment.ShipmentMoveJurisdictionMile{
			BusinessUnitID:    params.move.BusinessUnitID,
			OrganizationID:    params.move.OrganizationID,
			ShipmentMoveID:    params.move.ID,
			ShipmentID:        params.move.ShipmentID,
			CountryCode:       normalizeJurisdictionCountry(item.Country),
			JurisdictionCode:  code,
			Sequence:          len(rows),
			Distance:          roundDistance(item.Distance),
			DistanceUnits:     units,
			Loaded:            params.move.Loaded,
			Source:            shipment.JurisdictionMileSourceRouteCalculation,
			Provider:          params.provider,
			DataVersion:       params.dataVersion,
			DistanceProfileID: pulid.ID(params.profileID),
			CalculatedAt:      params.calculatedAt,
		}
		if item.Toll > 0 {
			toll := roundDistance(item.Toll)
			row.TollDistance = &toll
		}
		if item.Ferry > 0 {
			ferry := roundDistance(item.Ferry)
			row.FerryDistance = &ferry
		}
		rows = append(rows, row)
	}
	return rows
}

func normalizeJurisdictionUnits(units string) string {
	if strings.EqualFold(strings.TrimSpace(units), shipment.JurisdictionDistanceUnitsKilometers) {
		return shipment.JurisdictionDistanceUnitsKilometers
	}
	return shipment.JurisdictionDistanceUnitsMiles
}

func normalizeJurisdictionCountry(country string) string {
	code := strings.ToUpper(strings.TrimSpace(country))
	if len(code) == 3 {
		return countryutils.ISO3ToISO2OrDefault(code, countryutils.ISO2UnitedStates)
	}
	if len(code) != 2 {
		return countryutils.ISO2UnitedStates
	}
	return code
}

func jurisdictionSumIn(rows []*shipment.ShipmentMoveJurisdictionMile, units string) float64 {
	target := normalizeJurisdictionUnits(units)
	sum := 0.0
	for _, row := range rows {
		if row == nil {
			continue
		}
		sum += storedmileage.ConvertDistance(row.Distance, row.DistanceUnits, target)
	}
	return roundDistance(sum)
}

func withinJurisdictionTolerance(distance, sum float64) bool {
	tolerance := math.Max(
		jurisdictionAbsoluteTolerance,
		math.Abs(distance)*jurisdictionRelativeTolerance,
	)
	return math.Abs(sum-distance) <= tolerance
}

func jurisdictionAttribution(
	distance float64,
	units string,
	rows []*shipment.ShipmentMoveJurisdictionMile,
) (string, float64) {
	if len(rows) == 0 {
		return JurisdictionAttributionUnattributed, 0
	}
	sum := jurisdictionSumIn(rows, units)
	if withinJurisdictionTolerance(distance, sum) {
		return JurisdictionAttributionAttributed, sum
	}
	return JurisdictionAttributionMismatch, sum
}

func jurisdictionMismatchWarning(
	distance float64,
	units string,
	rows []*shipment.ShipmentMoveJurisdictionMile,
) (string, bool) {
	attribution, sum := jurisdictionAttribution(distance, units, rows)
	if attribution != JurisdictionAttributionMismatch {
		return "", false
	}
	return fmt.Sprintf(
		"Jurisdiction breakdown totals %.2f %s but the route distance is %.2f %s",
		sum,
		normalizeJurisdictionUnits(units),
		roundDistance(distance),
		normalizeJurisdictionUnits(units),
	), true
}

func appendMetadataWarning(metadata map[string]any, warning string) map[string]any {
	if metadata == nil {
		metadata = make(map[string]any, 1)
	}
	existing, _ := metadata["warnings"].([]string)
	metadata["warnings"] = mergeWarnings(existing, []string{warning})
	return metadata
}

func mergeWarnings(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	merged := make([]string, 0, len(base)+len(extra))
	merged = append(merged, base...)
	return append(merged, extra...)
}

func jurisdictionMileResults(
	rows []*shipment.ShipmentMoveJurisdictionMile,
) []services.JurisdictionMileResult {
	if len(rows) == 0 {
		return nil
	}
	results := make([]services.JurisdictionMileResult, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		results = append(results, services.JurisdictionMileResult{
			CountryCode:      row.CountryCode,
			JurisdictionCode: row.JurisdictionCode,
			Distance:         row.Distance,
			DistanceUnits:    row.DistanceUnits,
			Loaded:           row.Loaded,
		})
	}
	return results
}

func applyManualDistance(
	move *shipment.ShipmentMove,
	signature string,
	calculatedAt int64,
) float64 {
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

func moveResult(
	move *shipment.ShipmentMove,
	idx int,
	warnings []string,
) services.DistanceMoveResult {
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
		JurisdictionMiles:   jurisdictionMileResults(move.JurisdictionMiles),
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
