package internaledilifecycle

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopactual"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ActorLabel = "Internal EDI"

var ErrStopActualsUnavailable = errors.New(
	"internal EDI lifecycle sync has no stop actual reconciler",
)

type ShipmentRepository interface {
	GetByID(
		ctx context.Context,
		req *shipmentports.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
}

type StopActualReconciler interface {
	ReconcileStopActuals(
		ctx context.Context,
		req *shipmentmoveports.ReconcileStopActualsRequest,
	) (*shipmentmoveports.ReconcileStopActualsResult, error)
}

type Applier struct {
	shipmentRepo ShipmentRepository
	coordinator  *shipmentstate.Coordinator
	stopActuals  StopActualReconciler
}

type Params struct {
	ShipmentRepo ShipmentRepository
	Coordinator  *shipmentstate.Coordinator
	StopActuals  StopActualReconciler
}

type ApplyRequest struct {
	Plan          *Plan
	SourceEventID string
	CorrelationID string
}

type PrepareRequest struct {
	Link      *edisync.ShipmentLink
	Direction edisync.TransferChangeDirection
}

type PrepareLoadedRequest struct {
	Link      *edisync.ShipmentLink
	Direction edisync.TransferChangeDirection
	Source    *shipment.Shipment
	Target    *shipment.Shipment
}

type Plan struct {
	Link             *edisync.ShipmentLink
	Direction        edisync.TransferChangeDirection
	Source           *shipment.Shipment
	Target           *shipment.Shipment
	Changed          *shipment.Shipment
	OppositeOriginal *shipment.Shipment
	OppositePrepared *shipment.Shipment
	OppositeInfo     pagination.TenantInfo
	Diffs            []StopActualDiff
	Conflicts        []Conflict
	ConflictReason   string
}

type StopActualDiff struct {
	MoveSequence            int64                  `json:"moveSequence"`
	StopSequence            int64                  `json:"stopSequence"`
	StopType                shipmentvocab.StopType `json:"stopType"`
	SourceStopID            string                 `json:"sourceStopId"`
	TargetStopID            string                 `json:"targetStopId"`
	PreviousActualArrival   *int64                 `json:"previousActualArrival,omitempty"`
	NewActualArrival        *int64                 `json:"newActualArrival,omitempty"`
	PreviousActualDeparture *int64                 `json:"previousActualDeparture,omitempty"`
	NewActualDeparture      *int64                 `json:"newActualDeparture,omitempty"`
}

type Conflict struct {
	Reason       string                 `json:"reason"`
	MoveSequence int64                  `json:"moveSequence"`
	StopSequence int64                  `json:"stopSequence"`
	StopType     shipmentvocab.StopType `json:"stopType"`
	SourceStopID string                 `json:"sourceStopId,omitempty"`
	TargetStopID string                 `json:"targetStopId,omitempty"`
	Field        string                 `json:"field,omitempty"`
	Message      string                 `json:"message,omitempty"`
}

type stopKey struct {
	moveSequence int64
	stopSequence int64
	stopType     shipmentvocab.StopType
}

type indexedStop struct {
	stop *shipment.Stop
}

type stopIndex struct {
	stops      map[stopKey]*indexedStop
	ambiguous  map[stopKey]Conflict
	actualKeys map[stopKey]struct{}
}

func New(p Params) *Applier {
	coordinator := p.Coordinator
	if coordinator == nil {
		coordinator = shipmentstate.NewCoordinator()
	}

	return &Applier{
		shipmentRepo: p.ShipmentRepo,
		coordinator:  coordinator,
		stopActuals:  p.StopActuals,
	}
}

func (a *Applier) Prepare(ctx context.Context, req PrepareRequest) (*Plan, error) {
	source, target, err := a.loadShipments(ctx, req.Link)
	if err != nil {
		return nil, err
	}

	return a.PrepareLoaded(PrepareLoadedRequest{
		Link:      req.Link,
		Direction: req.Direction,
		Source:    source,
		Target:    target,
	})
}

func (a *Applier) PrepareLoaded(req PrepareLoadedRequest) (*Plan, error) {
	plan := basePlan(req)
	if plan == nil {
		return nil, nil
	}

	plan.OppositePrepared = cloneShipment(plan.OppositeOriginal)
	diffs, conflicts := applyStopActuals(plan.Changed, plan.OppositePrepared)
	plan.Diffs = diffs
	plan.Conflicts = conflicts
	if len(conflicts) > 0 {
		plan.ConflictReason = "Could not map all linked shipment stop actuals"
		return plan, nil
	}

	if len(diffs) == 0 {
		return plan, nil
	}

	if err := a.coordinator.PrepareForUpdate(
		plan.OppositeOriginal,
		plan.OppositePrepared,
	); err != nil {
		multiErr, refused := errortypes.RefusalOf(err)
		if !refused {
			return nil, err
		}
		plan.ConflictReason = multiErr.Error()
		plan.Conflicts = make([]Conflict, 0, len(multiErr.Errors))
		for _, fieldErr := range multiErr.Errors {
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Reason:  "coordinatorValidation",
				Field:   fieldErr.Field(),
				Message: fieldErr.Message,
			})
		}
	}

	return plan, nil
}

func (a *Applier) ApplyPrepared(ctx context.Context, req ApplyRequest) (*shipment.Shipment, error) {
	plan := req.Plan
	if plan == nil || plan.OppositePrepared == nil {
		return nil, nil
	}
	if a.stopActuals == nil {
		return nil, ErrStopActualsUnavailable
	}

	origin := &shipmentmoveports.StopActualEventOrigin{
		ActorLabel:    ActorLabel,
		CorrelationID: req.CorrelationID,
		Metadata: map[string]any{
			edisync.InternalEDIMirroredFromEventIDKey: req.SourceEventID,
			edisync.InternalEDIShipmentLinkIDKey:      plan.Link.ID.String(),
		},
	}
	for _, target := range syncedMoveTargets(plan) {
		if _, err := a.stopActuals.ReconcileStopActuals(
			ctx,
			&shipmentmoveports.ReconcileStopActualsRequest{
				TenantInfo: plan.OppositeInfo,
				MoveID:     target.moveID,
				Plan:       target.plan,
				Origin:     origin,
			},
		); err != nil {
			return nil, err
		}
	}

	return a.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         plan.OppositeOriginal.ID,
		TenantInfo: plan.OppositeInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
}

type moveTarget struct {
	moveID   pulid.ID
	sequence int64
	plan     shipmentmoveports.StopActualPlanner
}

func syncedMoveTargets(plan *Plan) []moveTarget {
	original := make(map[pulid.ID]*shipment.Stop)
	for _, move := range plan.OppositeOriginal.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop != nil {
				original[stop.ID] = stop
			}
		}
	}

	targets := make([]moveTarget, 0, len(plan.OppositePrepared.Moves))
	for _, move := range plan.OppositePrepared.Moves {
		if move == nil {
			continue
		}
		wanted := make(map[pulid.ID]stopactual.Change)
		for _, stop := range move.Stops {
			if stop == nil || actualsMatch(stop, original[stop.ID]) {
				continue
			}
			wanted[stop.ID] = stopactual.Change{
				StopID:    stop.ID,
				Arrival:   cloneInt64(stop.ActualArrival),
				Departure: cloneInt64(stop.ActualDeparture),
			}
		}
		if len(wanted) == 0 {
			continue
		}
		targets = append(targets, moveTarget{
			moveID:   move.ID,
			sequence: move.Sequence,
			plan:     planTowards(wanted),
		})
	}
	slices.SortStableFunc(targets, func(left, right moveTarget) int {
		return cmp.Compare(left.sequence, right.sequence)
	})

	return targets
}

func planTowards(wanted map[pulid.ID]stopactual.Change) shipmentmoveports.StopActualPlanner {
	return func(_ context.Context, move *shipment.ShipmentMove) ([]stopactual.Change, error) {
		changes := make([]stopactual.Change, 0, len(wanted))
		for _, stop := range move.ActiveStopsBySequence() {
			want, ok := wanted[stop.ID]
			if !ok {
				continue
			}
			change := stopactual.Change{StopID: stop.ID}
			if want.Arrival != nil && !int64PtrEqual(want.Arrival, stop.ActualArrival) {
				change.Arrival = cloneInt64(want.Arrival)
			}
			if want.Departure != nil && !int64PtrEqual(want.Departure, stop.ActualDeparture) {
				change.Departure = cloneInt64(want.Departure)
			}
			if !change.Empty() {
				changes = append(changes, change)
			}
		}
		return changes, nil
	}
}

func (a *Applier) loadShipments(
	ctx context.Context,
	link *edisync.ShipmentLink,
) (*shipment.Shipment, *shipment.Shipment, error) {
	source, err := a.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: link.SourceShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.SourceOrganizationID,
			BuID:  link.BusinessUnitID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	target, err := a.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: link.TargetShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.TargetOrganizationID,
			BuID:  link.BusinessUnitID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return source, target, nil
}

func basePlan(req PrepareLoadedRequest) *Plan {
	if req.Link == nil || req.Source == nil || req.Target == nil {
		return nil
	}

	plan := &Plan{
		Link:      req.Link,
		Direction: req.Direction,
		Source:    req.Source,
		Target:    req.Target,
	}
	if req.Direction == edisync.TransferChangeDirectionSourceToTarget {
		plan.Changed = req.Source
		plan.OppositeOriginal = req.Target
		plan.OppositeInfo = pagination.TenantInfo{
			OrgID: req.Link.TargetOrganizationID,
			BuID:  req.Link.BusinessUnitID,
		}
		return plan
	}

	plan.Changed = req.Target
	plan.OppositeOriginal = req.Source
	plan.OppositeInfo = pagination.TenantInfo{
		OrgID: req.Link.SourceOrganizationID,
		BuID:  req.Link.BusinessUnitID,
	}
	return plan
}

func applyStopActuals(
	changed *shipment.Shipment,
	opposite *shipment.Shipment,
) ([]StopActualDiff, []Conflict) {
	changedIndex := indexStops(changed)
	oppositeIndex := indexStops(opposite)
	conflicts := make([]Conflict, 0)

	diffs := make([]StopActualDiff, 0)
	for key := range changedIndex.actualKeys {
		changedStop := changedIndex.stops[key]
		if changedConflict, ok := changedIndex.ambiguous[key]; ok {
			conflicts = append(conflicts, changedConflict)
			continue
		}

		if oppositeConflict, ok := oppositeIndex.ambiguous[key]; ok {
			conflicts = append(conflicts, oppositeConflict)
			continue
		}

		oppositeStop := oppositeIndex.stops[key]
		if oppositeStop == nil {
			conflicts = append(conflicts, Conflict{
				Reason:       "missingStopMatch",
				MoveSequence: key.moveSequence,
				StopSequence: key.stopSequence,
				StopType:     key.stopType,
				SourceStopID: changedStop.stop.ID.String(),
			})
			continue
		}

		arrival := mergedActual(changedStop.stop.ActualArrival, oppositeStop.stop.ActualArrival)
		departure := mergedActual(
			changedStop.stop.ActualDeparture,
			oppositeStop.stop.ActualDeparture,
		)
		if int64PtrEqual(arrival, oppositeStop.stop.ActualArrival) &&
			int64PtrEqual(departure, oppositeStop.stop.ActualDeparture) {
			continue
		}

		diff := StopActualDiff{
			MoveSequence:            key.moveSequence,
			StopSequence:            key.stopSequence,
			StopType:                key.stopType,
			SourceStopID:            changedStop.stop.ID.String(),
			TargetStopID:            oppositeStop.stop.ID.String(),
			PreviousActualArrival:   cloneInt64(oppositeStop.stop.ActualArrival),
			NewActualArrival:        cloneInt64(arrival),
			PreviousActualDeparture: cloneInt64(oppositeStop.stop.ActualDeparture),
			NewActualDeparture:      cloneInt64(departure),
		}
		oppositeStop.stop.ActualArrival = arrival
		oppositeStop.stop.ActualDeparture = departure
		diffs = append(diffs, diff)
	}

	return diffs, conflicts
}

func indexStops(entity *shipment.Shipment) stopIndex {
	result := stopIndex{
		stops:      make(map[stopKey]*indexedStop),
		ambiguous:  make(map[stopKey]Conflict),
		actualKeys: make(map[stopKey]struct{}),
	}
	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}

			key := stopKey{
				moveSequence: move.Sequence,
				stopSequence: stop.Sequence,
				stopType:     stop.Type,
			}
			if hasActualSignal(stop) {
				result.actualKeys[key] = struct{}{}
			}

			if existing := result.stops[key]; existing != nil {
				if hasActualSignal(existing.stop) || hasActualSignal(stop) {
					result.actualKeys[key] = struct{}{}
				}
				result.ambiguous[key] = Conflict{
					Reason:       "ambiguousStopMatch",
					MoveSequence: key.moveSequence,
					StopSequence: key.stopSequence,
					StopType:     key.stopType,
					SourceStopID: existing.stop.ID.String(),
					TargetStopID: stop.ID.String(),
				}
				continue
			}

			result.stops[key] = &indexedStop{
				stop: stop,
			}
		}
	}

	return result
}

func cloneShipment(entity *shipment.Shipment) *shipment.Shipment {
	if entity == nil {
		return nil
	}

	clone := *entity
	clone.Moves = make([]*shipment.ShipmentMove, 0, len(entity.Moves))
	for _, move := range entity.Moves {
		if move == nil {
			clone.Moves = append(clone.Moves, nil)
			continue
		}

		moveClone := *move
		moveClone.Stops = make([]*shipment.Stop, 0, len(move.Stops))
		for _, stop := range move.Stops {
			if stop == nil {
				moveClone.Stops = append(moveClone.Stops, nil)
				continue
			}

			stopClone := *stop
			stopClone.ActualArrival = cloneInt64(stop.ActualArrival)
			stopClone.ActualDeparture = cloneInt64(stop.ActualDeparture)
			moveClone.Stops = append(moveClone.Stops, &stopClone)
		}
		clone.Moves = append(clone.Moves, &moveClone)
	}

	return &clone
}

func hasActualSignal(stop *shipment.Stop) bool {
	return stop != nil && (stop.ActualArrival != nil || stop.ActualDeparture != nil)
}

func actualsMatch(prepared, original *shipment.Stop) bool {
	return original != nil &&
		int64PtrEqual(prepared.ActualArrival, original.ActualArrival) &&
		int64PtrEqual(prepared.ActualDeparture, original.ActualDeparture)
}

func mergedActual(changed, opposite *int64) *int64 {
	if changed == nil {
		return cloneInt64(opposite)
	}

	return cloneInt64(changed)
}

func int64PtrEqual(left *int64, right *int64) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}

	clone := *value
	return &clone
}
