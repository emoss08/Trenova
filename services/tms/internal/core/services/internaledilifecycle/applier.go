package internaledilifecycle

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentRepository interface {
	GetByID(ctx context.Context, req *repositories.GetShipmentByIDRequest) (*shipment.Shipment, error)
	UpdateOperationalLifecycle(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*shipment.Shipment, error)
}

type Applier struct {
	shipmentRepo ShipmentRepository
	coordinator  *shipmentstate.Coordinator
}

type Params struct {
	ShipmentRepo ShipmentRepository
	Coordinator  *shipmentstate.Coordinator
}

type PrepareRequest struct {
	Link      *edi.ShipmentLink
	Direction edi.TransferChangeDirection
}

type PrepareLoadedRequest struct {
	Link      *edi.ShipmentLink
	Direction edi.TransferChangeDirection
	Source    *shipment.Shipment
	Target    *shipment.Shipment
}

type Plan struct {
	Link             *edi.ShipmentLink
	Direction        edi.TransferChangeDirection
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
	MoveSequence            int64             `json:"moveSequence"`
	StopSequence            int64             `json:"stopSequence"`
	StopType                shipment.StopType `json:"stopType"`
	SourceStopID            string            `json:"sourceStopId"`
	TargetStopID            string            `json:"targetStopId"`
	PreviousActualArrival   *int64            `json:"previousActualArrival,omitempty"`
	NewActualArrival        *int64            `json:"newActualArrival,omitempty"`
	PreviousActualDeparture *int64            `json:"previousActualDeparture,omitempty"`
	NewActualDeparture      *int64            `json:"newActualDeparture,omitempty"`
}

type Conflict struct {
	Reason       string            `json:"reason"`
	MoveSequence int64             `json:"moveSequence"`
	StopSequence int64             `json:"stopSequence"`
	StopType     shipment.StopType `json:"stopType"`
	SourceStopID string            `json:"sourceStopId,omitempty"`
	TargetStopID string            `json:"targetStopId,omitempty"`
	Field        string            `json:"field,omitempty"`
	Message      string            `json:"message,omitempty"`
}

type stopKey struct {
	moveSequence int64
	stopSequence int64
	stopType     shipment.StopType
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

	if multiErr := a.coordinator.PrepareForUpdate(
		plan.OppositeOriginal,
		plan.OppositePrepared,
	); multiErr != nil {
		plan.ConflictReason = multiErr.Error()
		plan.Conflicts = make([]Conflict, 0, len(multiErr.Errors))
		for _, err := range multiErr.Errors {
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Reason:  "coordinatorValidation",
				Field:   err.Field,
				Message: err.Message,
			})
		}
	}

	return plan, nil
}

func (a *Applier) ApplyPrepared(ctx context.Context, plan *Plan) (*shipment.Shipment, error) {
	if plan == nil || plan.OppositePrepared == nil {
		return nil, nil
	}

	return a.shipmentRepo.UpdateOperationalLifecycle(ctx, plan.OppositePrepared)
}

func (a *Applier) loadShipments(
	ctx context.Context,
	link *edi.ShipmentLink,
) (*shipment.Shipment, *shipment.Shipment, error) {
	source, err := a.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: link.SourceShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.SourceOrganizationID,
			BuID:  link.BusinessUnitID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	target, err := a.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: link.TargetShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: link.TargetOrganizationID,
			BuID:  link.BusinessUnitID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
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
	if req.Direction == edi.TransferChangeDirectionSourceToTarget {
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

		if actualsEqual(changedStop.stop, oppositeStop.stop) {
			continue
		}

		diff := StopActualDiff{
			MoveSequence:            key.moveSequence,
			StopSequence:            key.stopSequence,
			StopType:                key.stopType,
			SourceStopID:            changedStop.stop.ID.String(),
			TargetStopID:            oppositeStop.stop.ID.String(),
			PreviousActualArrival:   cloneInt64(oppositeStop.stop.ActualArrival),
			NewActualArrival:        cloneInt64(changedStop.stop.ActualArrival),
			PreviousActualDeparture: cloneInt64(oppositeStop.stop.ActualDeparture),
			NewActualDeparture:      cloneInt64(changedStop.stop.ActualDeparture),
		}
		oppositeStop.stop.ActualArrival = cloneInt64(changedStop.stop.ActualArrival)
		oppositeStop.stop.ActualDeparture = cloneInt64(changedStop.stop.ActualDeparture)
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

func actualsEqual(left *shipment.Stop, right *shipment.Stop) bool {
	if left == nil || right == nil {
		return left == right
	}

	return int64PtrEqual(left.ActualArrival, right.ActualArrival) &&
		int64PtrEqual(left.ActualDeparture, right.ActualDeparture)
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
