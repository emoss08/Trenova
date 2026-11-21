package distancecalculationservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/domain/storedmileage"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pcmiler"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var measuredPurposes = []string{
	distancecontrol.PurposePay,
	distancecontrol.PurposeBilling,
	distancecontrol.PurposeFuel,
}

var purposeLabels = map[string]string{
	distancecontrol.PurposePay:     "Pay",
	distancecontrol.PurposeBilling: "Billing",
	distancecontrol.PurposeFuel:    "Fuel",
}

type purposeGroup struct {
	profileID pulid.ID
	purposes  []string
}

func (g purposeGroup) lead() string {
	for _, purpose := range g.purposes {
		if purpose == distancecontrol.PurposeFuel {
			return purpose
		}
	}
	return g.purposes[0]
}

func separatePurposeGroups(
	control *distancecontrol.DistanceControl,
	operationalPurpose string,
) []purposeGroup {
	if control == nil {
		return nil
	}
	operational := control.ProfileIDForPurpose(operationalPurpose)
	groups := make([]purposeGroup, 0, len(measuredPurposes))
	for _, purpose := range measuredPurposes {
		profileID := control.ProfileIDForPurpose(purpose)
		if profileID == operational {
			continue
		}
		grouped := false
		for i := range groups {
			if groups[i].profileID == profileID {
				groups[i].purposes = append(groups[i].purposes, purpose)
				grouped = true
				break
			}
		}
		if !grouped {
			groups = append(groups, purposeGroup{profileID: profileID, purposes: []string{purpose}})
		}
	}
	return groups
}

func fuelIsSeparate(control *distancecontrol.DistanceControl, move *shipment.ShipmentMove) bool {
	return control != nil &&
		control.FuelDistanceProfileID != control.ProfileIDForPurpose(movePurpose(move))
}

func routedSource(source string) bool {
	return source == distancecalculation.SourcePCMiler ||
		source == distancecalculation.SourceStoredMileage
}

func separateFuelDistance(
	control *distancecontrol.DistanceControl,
	move *shipment.ShipmentMove,
	distance float64,
	units string,
) (float64, bool) {
	if !fuelIsSeparate(control, move) || !routedSource(move.DistanceSource) {
		return 0, false
	}
	return roundDistance(storedmileage.ConvertDistance(distance, units, move.DistanceUnits)), true
}

func (s *Service) storeFuelDistance(
	ctx context.Context,
	control *distancecontrol.DistanceControl,
	move *shipment.ShipmentMove,
	distance float64,
	options pcmiler.RouteOptions,
) error {
	fuelDistance, separate := separateFuelDistance(control, move, distance, options.DistanceUnits)
	switch {
	case separate:
		move.FuelDistance = &fuelDistance
	case move.FuelDistance != nil:
		move.FuelDistance = nil
	default:
		return nil
	}
	return s.distanceCalcRepo.UpdateMoveDistance(ctx, move)
}

type purposeDistance struct {
	distance        float64
	units           string
	source          string
	provider        string
	dataVersion     string
	profile         *distanceprofile.DistanceProfile
	storedMileageID pulid.ID
	jurisdictions   []pcmiler.JurisdictionDistance
}

type purposeTarget struct {
	move     *shipment.ShipmentMove
	purposes []string
	profile  *distanceprofile.DistanceProfile
	options  pcmiler.RouteOptions
}

type movePurposes struct {
	fuelSeparate bool
	pending      map[string]struct{}
	values       map[string]purposeDistance
}

type purposePlan struct {
	moves   map[*shipment.ShipmentMove]*movePurposes
	targets map[string]purposeTarget
}

func newPurposePlan() *purposePlan {
	return &purposePlan{
		moves:   make(map[*shipment.ShipmentMove]*movePurposes),
		targets: make(map[string]purposeTarget),
	}
}

func (p *purposePlan) forMove(move *shipment.ShipmentMove) *movePurposes {
	entry, ok := p.moves[move]
	if !ok {
		entry = &movePurposes{
			pending: make(map[string]struct{}, len(measuredPurposes)),
			values:  make(map[string]purposeDistance, len(measuredPurposes)),
		}
		p.moves[move] = entry
	}
	return entry
}

func (p *purposePlan) record(
	move *shipment.ShipmentMove,
	purposes []string,
	value purposeDistance,
) {
	entry := p.forMove(move)
	for _, purpose := range purposes {
		entry.values[purpose] = value
	}
}

type purposeRequest struct {
	plan        *purposePlan
	entity      *shipment.Shipment
	move        *shipment.ShipmentMove
	index       int
	signature   string
	control     *distancecontrol.DistanceControl
	hazmatTypes []string
	runtimes    map[string]pcmilerRuntime
}

func (s *Service) queuePurposeDistances(
	ctx context.Context,
	req *purposeRequest,
	requests []pcmiler.RouteRequest,
) ([]pcmiler.RouteRequest, error) {
	groups := separatePurposeGroups(req.control, movePurpose(req.move))
	entry := req.plan.forMove(req.move)
	entry.fuelSeparate = fuelIsSeparate(req.control, req.move)
	for _, group := range groups {
		for _, purpose := range group.purposes {
			entry.pending[purpose] = struct{}{}
		}
		runtime := s.runtimeForPurpose(
			ctx,
			req.entity,
			req.control,
			group.lead(),
			req.hazmatTypes,
			req.runtimes,
		)
		if runtime.profile == nil {
			continue
		}
		stored, storedOK, err := s.storedMileage(
			ctx,
			req.entity,
			req.move,
			req.control,
			runtime.profile,
			runtime.options,
			req.hazmatTypes,
		)
		if err != nil {
			return nil, err
		}
		if storedOK && useStoredMileage(stored, runtime) {
			req.plan.record(req.move, group.purposes, purposeDistance{
				distance:        stored.Distance,
				units:           runtime.options.DistanceUnits,
				source:          distancecalculation.SourceStoredMileage,
				provider:        stored.Provider,
				dataVersion:     stored.DataVersion,
				profile:         runtime.profile,
				storedMileageID: stored.ID,
				jurisdictions:   pcmilerJurisdictionsFromStored(stored.JurisdictionDistances),
			})
			s.incrementStoredMileageHit(ctx, req.entity, stored.ID)
			continue
		}
		if !runtime.ready {
			continue
		}
		routeID := req.signature + "#" + group.lead() + "#" + strconv.Itoa(req.index)
		route, ok := pcmilerRoute(req.move, runtime.options, routeID)
		if !ok {
			continue
		}
		req.plan.targets[routeID] = purposeTarget{
			move:     req.move,
			purposes: group.purposes,
			profile:  runtime.profile,
			options:  runtime.options,
		}
		requests = append(requests, route)
	}
	return requests, nil
}

func (s *Service) recordPurposeResult(
	ctx context.Context,
	plan *purposePlan,
	entity *shipment.Shipment,
	control *distancecontrol.DistanceControl,
	hazmatTypes []string,
	result pcmiler.RouteMileage,
) bool {
	target, ok := plan.targets[result.RouteID]
	if !ok {
		return false
	}
	plan.record(target.move, target.purposes, purposeDistance{
		distance:      result.Distance,
		units:         target.options.DistanceUnits,
		source:        distancecalculation.SourcePCMiler,
		provider:      string(integration.TypePCMiler),
		dataVersion:   target.options.DataVersion,
		profile:       target.profile,
		jurisdictions: result.JurisdictionDistances,
	})
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
	return true
}

func (p *purposePlan) finish(
	moves []*shipment.ShipmentMove,
	resp *shipmentdistanceports.DistanceCalculationResponse,
) {
	warningsByMove := make(map[*shipment.ShipmentMove][]string, len(p.moves))
	for _, move := range moves {
		if move == nil {
			continue
		}
		entry, ok := p.moves[move]
		if !ok || !routedSource(move.DistanceSource) {
			move.ClearPurposeDistances()
			continue
		}
		warningsByMove[move] = entry.apply(move)
	}

	for i := range resp.Moves {
		result := &resp.Moves[i]
		if result.MoveIndex < 0 || result.MoveIndex >= len(moves) {
			continue
		}
		move := moves[result.MoveIndex]
		if move == nil {
			continue
		}
		result.PayDistance = move.PayDistance
		result.BillingDistance = move.BillingDistance
		result.FuelDistance = move.FuelDistance
		result.JurisdictionMiles = jurisdictionMileResults(move.JurisdictionMiles)
		result.Warnings = mergeWarnings(result.Warnings, warningsByMove[move])
	}
}

func (e *movePurposes) apply(move *shipment.ShipmentMove) []string {
	move.ClearPurposeDistances()
	var warnings []string
	summary := make(map[string]any, len(e.values))
	for _, purpose := range measuredPurposes {
		if _, wanted := e.pending[purpose]; !wanted {
			continue
		}
		value, ok := e.values[purpose]
		if !ok {
			warnings = append(warnings, purposeLabels[purpose]+
				" distance could not be routed with its distance profile, so the move's distance stands for it")
			continue
		}
		distance := roundDistance(
			storedmileage.ConvertDistance(value.distance, value.units, move.DistanceUnits),
		)
		setPurposeDistance(move, purpose, distance)
		item := map[string]any{
			"distance":            distance,
			"source":              value.source,
			"distanceProfileId":   value.profile.ID.String(),
			"distanceProfileName": value.profile.Name,
		}
		if !value.storedMileageID.IsNil() {
			item["storedMileageId"] = value.storedMileageID.String()
		}
		summary[purpose] = item
	}

	if e.fuelSeparate {
		warnings = append(warnings, e.applyFuelJurisdictions(move)...)
	}

	if len(summary) > 0 || len(warnings) > 0 {
		if move.DistanceMetadata == nil {
			move.DistanceMetadata = make(map[string]any, 1)
		}
		if len(summary) > 0 {
			move.DistanceMetadata["purposeDistances"] = summary
		}
		for _, warning := range warnings {
			move.DistanceMetadata = appendMetadataWarning(move.DistanceMetadata, warning)
		}
	}
	return warnings
}

func (e *movePurposes) applyFuelJurisdictions(move *shipment.ShipmentMove) []string {
	value, ok := e.values[distancecontrol.PurposeFuel]
	if !ok {
		move.JurisdictionMiles = []*shipment.ShipmentMoveJurisdictionMile{}
		move.JurisdictionMilesDirty = true
		return nil
	}
	rows := buildJurisdictionMiles(moveDistanceParams{
		move:          move,
		distance:      value.distance,
		source:        value.source,
		provider:      value.provider,
		dataVersion:   value.dataVersion,
		distanceUnit:  value.units,
		profileID:     value.profile.ID.String(),
		calculatedAt:  calculatedAtOf(move),
		jurisdictions: value.jurisdictions,
	})
	move.JurisdictionMiles = rows
	move.JurisdictionMilesDirty = true
	if warning, mismatch := jurisdictionMismatchWarning(value.distance, value.units, rows); mismatch {
		return []string{warning}
	}
	return nil
}

func calculatedAtOf(move *shipment.ShipmentMove) int64 {
	if move.DistanceCalculatedAt != nil {
		return *move.DistanceCalculatedAt
	}
	return timeutils.NowUnix()
}

func setPurposeDistance(move *shipment.ShipmentMove, purpose string, distance float64) {
	switch purpose {
	case distancecontrol.PurposePay:
		move.PayDistance = &distance
	case distancecontrol.PurposeBilling:
		move.BillingDistance = &distance
	case distancecontrol.PurposeFuel:
		move.FuelDistance = &distance
	}
}
