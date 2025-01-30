package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"go.uber.org/fx"
)

type MoveValidatorParams struct {
	fx.In

	DB db.Connection
}

type MoveValidator struct {
	db db.Connection
}

func NewMoveValidator(p MoveValidatorParams) *MoveValidator {
	return &MoveValidator{
		db: p.DB,
	}
}

func (v *MoveValidator) Validate(ctx context.Context, valCtx *validator.ValidationContext, m *shipment.ShipmentMove, multiErr *errors.MultiError, idx int) {
	moveMultiErr := multiErr.WithIndex("moves", idx)

	m.Validate(ctx, moveMultiErr)
	v.validateStops(m, moveMultiErr)

	if valCtx.IsCreate {
		v.validateID(m, moveMultiErr)
	}
}

func (v *MoveValidator) validateID(m *shipment.ShipmentMove, multiErr *errors.MultiError) {
	if m.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

func (v *MoveValidator) validateStops(m *shipment.ShipmentMove, multiErr *errors.MultiError) {
	v.validateStopLength(m, multiErr)
	v.validateStopTimes(m, multiErr)
	v.validateStopSequence(m, multiErr)
}

// validateStopLength validates that atleast two stops are in a movement.
func (v *MoveValidator) validateStopLength(m *shipment.ShipmentMove, multiErr *errors.MultiError) {
	if len(m.Stops) < 2 {
		multiErr.Add("stops", errors.ErrInvalid, "At least two stops is required in a move")
		return
	}
}

func (v *MoveValidator) validateStopTimes(m *shipment.ShipmentMove, multiErr *errors.MultiError) {
	if len(m.Stops) <= 1 {
		return
	}

	for i := 0; i < len(m.Stops)-1; i++ {
		currStop := m.Stops[i]
		nextStop := m.Stops[i+1]

		// Validate individual stop times (arrival before departure at same stop)
		if currStop.PlannedArrival >= currStop.PlannedDeparture {
			multiErr.Add(
				fmt.Sprintf("stops[%d].plannedArrival", i),
				errors.ErrInvalid,
				"Planned arrival must be before planned departure",
			)
		}

		// Validate sequential stop times
		if currStop.PlannedDeparture >= nextStop.PlannedArrival {
			multiErr.Add(
				fmt.Sprintf("stops[%d].plannedDeparture", i),
				errors.ErrInvalid,
				"Planned departure must be before next stop's planned arrival",
			)
		}

		// Validate actual times if present
		if currStop.ActualArrival != nil && currStop.ActualDeparture != nil {
			if *currStop.ActualArrival >= *currStop.ActualDeparture {
				multiErr.Add(
					fmt.Sprintf("stops[%d].actualArrival", i),
					errors.ErrInvalid,
					"Actual arrival must be before actual departure",
				)
			}
		}

		if currStop.ActualDeparture != nil && nextStop.ActualArrival != nil {
			if *currStop.ActualDeparture >= *nextStop.ActualArrival {
				multiErr.Add(
					fmt.Sprintf("stops[%d].actualDeparture", i),
					errors.ErrInvalid,
					"Actual departure must be before next stop's actual arrival",
				)
			}
		}
	}
}

func (v *MoveValidator) validateStopSequence(m *shipment.ShipmentMove, multiErr *errors.MultiError) {
	// Quick lookup maps for stop types
	pickupTypes := map[shipment.StopType]bool{ //nolint: exhaustive // We only need to check for pickup and split pickup
		shipment.StopTypePickup:      true,
		shipment.StopTypeSplitPickup: true,
	}

	deliveryTypes := map[shipment.StopType]bool{ //nolint: exhaustive // We only need to check for delivery and split delivery
		shipment.StopTypeDelivery:      true,
		shipment.StopTypeSplitDelivery: true,
	}

	// Guard clause for empty stops
	if len(m.Stops) == 0 {
		multiErr.Add("stops", errors.ErrInvalid, "Movement must have at least one stop")
		return
	}

	// Validate first stop is a pickup type
	if !pickupTypes[m.Stops[0].Type] {
		multiErr.Add(
			"stops[0].type",
			errors.ErrInvalid,
			"First stop must be a pickup or split pickup",
		)
	}

	// Validate last stop is a delivery type
	if !deliveryTypes[m.Stops[len(m.Stops)-1].Type] {
		multiErr.Add(
			fmt.Sprintf("stops[%d].type", len(m.Stops)-1),
			errors.ErrInvalid,
			"Last stop must be a delivery or split delivery",
		)
	}

	// Keep track of all pickups before current stop
	hasPickup := false
	for i, stop := range m.Stops {
		// Validate stop type is allowed
		if !pickupTypes[stop.Type] && !deliveryTypes[stop.Type] {
			multiErr.Add(
				fmt.Sprintf("stops[%d].type", i),
				errors.ErrInvalid,
				"Stop type must be pickup or delivery",
			)
			continue
		}

		// Track pickup status and validate delivery sequence
		if pickupTypes[stop.Type] {
			hasPickup = true
		} else if deliveryTypes[stop.Type] && !hasPickup {
			multiErr.Add(
				fmt.Sprintf("stops[%d].type", i),
				errors.ErrInvalid,
				"Delivery stop must be preceded by a pickup or split pickup",
			)
		}
	}
}
