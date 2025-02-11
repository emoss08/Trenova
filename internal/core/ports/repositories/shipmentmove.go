package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
)

type GetMoveByIDOptions struct {
	// ID of the move
	MoveID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the business unit
	BuID pulid.ID

	// Expand move details (Optional)
	ExpandMoveDetails bool
}

type UpdateMoveStatusRequest struct {
	// Fetch the move
	GetMoveOpts GetMoveByIDOptions

	// Status of the move
	Status shipment.MoveStatus
}

type BulkUpdateMoveStatusRequest struct {
	// IDs of the moves
	MoveIDs []pulid.ID

	// Status of the move
	Status shipment.MoveStatus
}

type GetMovesByShipmentIDOptions struct {
	// ID of the shipment
	ShipmentID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the business unit
	BuID pulid.ID
}

type SplitQuantity struct {
	// Pieces to split
	Pieces *int `json:"pieces"`

	// Weight to split
	Weight *int `json:"weight"`
}

type SplitStopTimes struct {
	// Planned arrival time for the split stops
	PlannedArrival int64 `json:"plannedArrival"`

	// Planned departure time for the split stops
	PlannedDeparture int64 `json:"plannedDeparture"`
}

type SplitMoveRequest struct {
	// ID of the move
	MoveID pulid.ID `json:"moveId"`

	// ID of the organization
	OrgID pulid.ID `json:"organizationId"`

	// ID of the business unit
	BuID pulid.ID `json:"businessUnitId"`

	// Location where the split will occur
	SplitLocationID pulid.ID `json:"splitLocationId"`

	// Quantities to split
	SplitQuantities SplitQuantity `json:"splitQuantities"`

	// The sequence number after which to perform the split
	SplitAfterStopSequence int `json:"splitAfterStopSequence"`

	// Times for the split delivery stop
	SplitDeliveryTimes SplitStopTimes `json:"splitDeliveryTimes"`

	// Times for the split pickup stop
	SplitPickupTimes SplitStopTimes `json:"splitPickupTimes"`
}

func (smr *SplitMoveRequest) Validate(ctx context.Context, move *shipment.ShipmentMove) *errors.MultiError {
	me := errors.NewMultiError()

	err := validation.ValidateStructWithContext(ctx, smr,
		validation.Field(&smr.MoveID, validation.Required.Error("Move ID is required")),
		validation.Field(&smr.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&smr.BuID, validation.Required.Error("Business Unit ID is required")),
		validation.Field(&smr.SplitLocationID, validation.Required.Error("Split Location ID is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, me)
		}
	}

	smr.validateSequence(move, me)

	smr.validateTimes(move, me)

	if me.HasErrors() {
		return me
	}

	return nil
}

func (smr *SplitMoveRequest) validateTimes(move *shipment.ShipmentMove, multiErr *errors.MultiError) {
	if len(move.Stops) != 2 {
		multiErr.Add("stops", errors.ErrInvalid, "Move must have exactly two stops")
		return
	}

	originalPickup := move.Stops[0]
	originalDelivery := move.Stops[1]

	// Validate split delivery times
	if smr.SplitDeliveryTimes.PlannedArrival <= originalPickup.PlannedDeparture {
		multiErr.Add("splitDeliveryTimes.plannedArrival", errors.ErrInvalid,
			"Split delivery planned arrival must be after original pickup planned departure")
	}
	if smr.SplitDeliveryTimes.PlannedDeparture <= smr.SplitDeliveryTimes.PlannedArrival {
		multiErr.Add("splitDeliveryTimes.plannedDeparture", errors.ErrInvalid,
			"Split delivery planned departure must be after split delivery planned arrival")
	}

	// Validate split pickup times
	if smr.SplitPickupTimes.PlannedArrival <= smr.SplitDeliveryTimes.PlannedDeparture {
		multiErr.Add("splitPickupTimes.plannedArrival", errors.ErrInvalid,
			"Split pickup planned arrival must be after split delivery planned departure")
	}
	if smr.SplitPickupTimes.PlannedDeparture <= smr.SplitPickupTimes.PlannedArrival {
		multiErr.Add("splitPickupTimes.plannedDeparture", errors.ErrInvalid,
			"Split pickup planned departure must be after split pickup planned arrival")
	}
	if originalDelivery.PlannedArrival <= smr.SplitPickupTimes.PlannedDeparture {
		multiErr.Add("splitPickupTimes.plannedDeparture", errors.ErrInvalid,
			"Original delivery planned arrival must be after split pickup planned departure")
	}
}

// TODO(Wolfred): This is a temporary validation. We will need to move this into an actual validator
func (smr *SplitMoveRequest) validateSequence(move *shipment.ShipmentMove, multiErr *errors.MultiError) {
	if len(move.Stops) == 0 {
		multiErr.Add("stops", errors.ErrInvalid, "Move has no stops to split")
		return
	}

	// Get the maximum sequence number
	maxSequence := -1 // Start at -1 to handle 0-based sequences
	for _, stop := range move.Stops {
		if stop.Sequence > maxSequence {
			maxSequence = stop.Sequence
		}
	}

	// For a simple pickup-delivery move, we can only split after the pickup (sequence 0)
	if len(move.Stops) == 2 {
		// First stop should be pickup
		if move.Stops[0].Type != shipment.StopTypePickup {
			multiErr.Add("stops", errors.ErrInvalid, "First stop must be a pickup")
			return
		}

		// Second stop should be delivery
		if move.Stops[1].Type != shipment.StopTypeDelivery {
			multiErr.Add("stops", errors.ErrInvalid, "Second stop must be a delivery")
			return
		}

		// Can only split after the pickup (sequence 0)
		if smr.SplitAfterStopSequence != 0 {
			multiErr.Add("splitAfterStopSequence", errors.ErrInvalid,
				"For a simple pickup-delivery move, must split after the pickup (sequence 0)")
			return
		}
	}
}

type SplitMoveResponse struct {
	// The original move after splitting
	OriginalMove *shipment.ShipmentMove `json:"originalMove,omitempty"`

	// The newly created move
	NewMove *shipment.ShipmentMove `json:"newMove,omitempty"`
}

type ShipmentMoveRepository interface {
	GetByID(ctx context.Context, opts GetMoveByIDOptions) (*shipment.ShipmentMove, error)
	UpdateStatus(ctx context.Context, opts *UpdateMoveStatusRequest) (*shipment.ShipmentMove, error)
	GetMovesByShipmentID(ctx context.Context, opts GetMovesByShipmentIDOptions) ([]*shipment.ShipmentMove, error)
	BulkUpdateStatus(ctx context.Context, opts BulkUpdateMoveStatusRequest) ([]*shipment.ShipmentMove, error)
	BulkInsert(ctx context.Context, moves []*shipment.ShipmentMove) ([]*shipment.ShipmentMove, error)
	SplitMove(ctx context.Context, req *SplitMoveRequest) (*SplitMoveResponse, error)
}
