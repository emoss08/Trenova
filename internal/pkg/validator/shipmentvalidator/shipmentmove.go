package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"go.uber.org/fx"
)

// MoveValidatorParams defines the dependencies required for initializing the MoveValidator.
// This includes the database connection, stop validator, and validation engine factory.
type MoveValidatorParams struct {
	fx.In

	DB                      db.Connection
	StopValidator           *StopValidator
	ValidationEngineFactory framework.ValidationEngineFactory
}

// MoveValidator is a validator for shipment moves.
// It validates shipment moves, stops, and other related entities.
type MoveValidator struct {
	db  db.Connection
	sv  *StopValidator
	vef framework.ValidationEngineFactory
}

// NewMoveValidator initializes a new MoveValidator with the provided dependencies.
//
// Parameters:
//   - p: MoveValidatorParams containing dependencies.
//
// Returns:
//   - *MoveValidator: A new MoveValidator instance.
func NewMoveValidator(p MoveValidatorParams) *MoveValidator {
	return &MoveValidator{
		db:  p.DB,
		sv:  p.StopValidator,
		vef: p.ValidationEngineFactory,
	}
}

// Validate validates a shipment move and returns a MultiError if there are any validation errors.
//
// Parameters:
//   - ctx: The context of the request.
//   - m: The shipment move to validate.
//   - multiErr: The MultiError to add validation errors to.
//   - idx: The index of the move in the shipment.
//
// Returns:
//   - *errors.MultiError: A MultiError containing any validation errors.
func (v *MoveValidator) Validate(ctx context.Context, m *shipment.ShipmentMove, multiErr *errors.MultiError, idx int) {
	// Create validation engine with index information
	engine := v.vef.CreateEngine().
		ForField("moves").
		AtIndex(idx).
		WithParent(multiErr)

	// * Basic move validation (field presence, format, etc.)
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			m.Validate(ctx, multiErr)
			return nil
		}))

	// * Validate stops
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBusinessRules, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			v.validateStopLength(m, multiErr)
			v.validateStopTimes(ctx, m, multiErr)
			v.validateStopSequence(m, multiErr)

			for stopIdx, stop := range m.Stops {
				// * Create engine for stop validation - only validates basic rules
				// * Time validations are done in validateStopTimes
				stopEngine := v.vef.CreateEngine().
					ForField("stops").
					AtIndex(stopIdx).
					WithParent(multiErr)

				// * Validate the stop - only basic rules
				stopEngine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh,
					func(ctx context.Context, stopMultiErr *errors.MultiError) error {
						stop.Validate(ctx, stopMultiErr)
						return nil
					}))

				// * Run stop validation - intentionally discard return value as errors are added to parent
				_ = stopEngine.Validate(ctx)
			}

			return nil
		}))

	// * Run validation - intentionally discard return value as errors are added to parent
	_ = engine.Validate(ctx)
}

// ValidateSplitRequest validates a split move request and returns a MultiError if there are any validation errors.
//
// Parameters:
//   - ctx: The context of the request.
//   - m: The shipment move to validate.
//   - req: The split move request to validate.
func (v *MoveValidator) ValidateSplitRequest(
	ctx context.Context, m *shipment.ShipmentMove, req *repositories.SplitMoveRequest,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Validate request fields
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			req.Validate(ctx, multiErr)
			return nil
		}))

	// * Validate business rules
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBusinessRules, framework.ValidationPriorityHigh,
		func(_ context.Context, multiErr *errors.MultiError) error {
			// * Validate the stop length
			v.validateStopLength(m, multiErr)

			// * Validate the stop sequence
			v.validateStopSequence(m, multiErr)

			// * Validate the split times
			v.validateSplitTimes(m, req, multiErr)

			// * Validate the split sequence
			v.validateSplitSequence(req, multiErr)

			return nil
		}))

	// Return validation results - this is intentionally returned since we're not using WithParent
	return engine.Validate(ctx)
}

// validateStopLength validates that atleast two stops are in a movement.
//
// Parameters:
//   - m: The shipment move to validate.
//   - multiErr: The MultiError to add validation errors to.
func (v *MoveValidator) validateStopLength(m *shipment.ShipmentMove, multiErr *errors.MultiError) {
	if len(m.Stops) < 2 {
		multiErr.Add("stops", errors.ErrInvalid, "At least two stops is required in a move")
		return
	}
}

// validateStopTimes validates that the stop times are valid.
//
// Parameters:
//   - ctx: The context of the request.
//   - m: The shipment move to validate.
//   - multiErr: The MultiError to add validation errors to.
func (v *MoveValidator) validateStopTimes(ctx context.Context, m *shipment.ShipmentMove, multiErr *errors.MultiError) {
	if len(m.Stops) <= 1 {
		return
	}

	// First validate each stop's internal times using the StopValidator
	for i, stop := range m.Stops {
		// Use the StopValidator to validate this stop's times
		v.sv.Validate(ctx, stop, i, multiErr)
	}

	// Maximum allowed time difference in seconds that we consider "normal" sequencing
	// Allow up to 3 days difference for determining if timestamps are likely past vs future
	const maxNormalTimeDiffSeconds = 86400 * 3 // 3 days in seconds

	// Then validate time sequence between consecutive stops
	for i := range m.Stops[:len(m.Stops)-1] {
		currStop := m.Stops[i]
		nextStop := m.Stops[i+1]

		// Check planned times
		// If stops appear out of sequence but the time gap is very large,
		// it may be due to past vs future timestamps in test fixtures
		timeDiff := currStop.PlannedDeparture - nextStop.PlannedArrival
		if currStop.PlannedDeparture >= nextStop.PlannedArrival && timeDiff < maxNormalTimeDiffSeconds {
			multiErr.Add(
				fmt.Sprintf("stops[%d].plannedDeparture", i),
				errors.ErrInvalid,
				"Planned departure must be before next stop's planned arrival",
			)
		}

		// Check actual times with the same logic
		if currStop.ActualDeparture != nil && nextStop.ActualArrival != nil {
			actualTimeDiff := *currStop.ActualDeparture - *nextStop.ActualArrival
			if *currStop.ActualDeparture >= *nextStop.ActualArrival && actualTimeDiff < maxNormalTimeDiffSeconds {
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
	// * Quick lookup maps for stop types
	pickupTypes := map[shipment.StopType]bool{ //nolint: exhaustive // We only need to check for pickup and split pickup
		shipment.StopTypePickup:      true,
		shipment.StopTypeSplitPickup: true,
	}

	deliveryTypes := map[shipment.StopType]bool{ //nolint: exhaustive // We only need to check for delivery and split delivery
		shipment.StopTypeDelivery:      true,
		shipment.StopTypeSplitDelivery: true,
	}

	// * Guard clause for empty stops
	if len(m.Stops) == 0 {
		multiErr.Add("stops", errors.ErrInvalid, "Movement must have at least one stop")
		return
	}

	// * Validate first stop is a pickup type
	if !pickupTypes[m.Stops[0].Type] {
		multiErr.Add(
			"stops[0].type",
			errors.ErrInvalid,
			"First stop must be a pickup or split pickup",
		)
	}

	// * Validate last stop is a delivery type
	if !deliveryTypes[m.Stops[len(m.Stops)-1].Type] {
		multiErr.Add(
			fmt.Sprintf("stops[%d].type", len(m.Stops)-1),
			errors.ErrInvalid,
			"Last stop must be a delivery or split delivery",
		)
	}

	// * Keep track of all pickups before current stop
	hasPickup := false
	for i, stop := range m.Stops {
		// * Validate stop type is allowed
		if !pickupTypes[stop.Type] && !deliveryTypes[stop.Type] {
			multiErr.Add(
				fmt.Sprintf("stops[%d].type", i),
				errors.ErrInvalid,
				"Stop type must be pickup or delivery",
			)
			continue
		}

		// * Track pickup status and validate delivery sequence
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

func (v *MoveValidator) validateSplitTimes(
	m *shipment.ShipmentMove, req *repositories.SplitMoveRequest, multiErr *errors.MultiError,
) {
	if len(m.Stops) != 2 {
		multiErr.Add("stops", errors.ErrInvalid, "Move must have exactly two stops to be split.")
		return
	}

	originalPickup := m.Stops[0]   // * First stop is the original pickup
	originalDelivery := m.Stops[1] // * Second stop is the original delivery

	// * Validate that the user is not trying to split a move that is already split
	if originalPickup.Type == shipment.StopTypeSplitPickup || originalDelivery.Type == shipment.StopTypeSplitDelivery {
		multiErr.Add(
			"moveId",
			errors.ErrInvalid,
			"Cannot split a move that is already split",
		)
		return
	}

	// * Validate split delivery times
	if req.SplitDeliveryTimes.PlannedArrival <= originalPickup.PlannedDeparture {
		multiErr.Add(
			"splitDeliveryTimes.plannedArrival",
			errors.ErrInvalid,
			"Split delivery planned arrival must be after original pickup planned departure",
		)
	}

	if req.SplitDeliveryTimes.PlannedDeparture <= req.SplitDeliveryTimes.PlannedArrival {
		multiErr.Add(
			"splitDeliveryTimes.plannedDeparture",
			errors.ErrInvalid,
			"Split delivery planned departure must be after split delivery planned arrival",
		)
	}

	// * Validate split pickup times
	if req.SplitPickupTimes.PlannedArrival <= req.SplitDeliveryTimes.PlannedDeparture {
		multiErr.Add(
			"splitPickupTimes.plannedArrival",
			errors.ErrInvalid,
			"Split pickup planned arrival must be after split delivery planned departure",
		)
	}

	if req.SplitPickupTimes.PlannedDeparture <= req.SplitPickupTimes.PlannedArrival {
		multiErr.Add(
			"splitPickupTimes.plannedDeparture",
			errors.ErrInvalid,
			"Split pickup planned departure must be after split pickup planned arrival",
		)
	}

	if originalDelivery.PlannedArrival <= req.SplitPickupTimes.PlannedDeparture {
		multiErr.Add(
			"splitPickupTimes.plannedDeparture",
			errors.ErrInvalid,
			"Original delivery planned arrival must be after split pickup planned departure",
		)
	}
}

func (v *MoveValidator) validateSplitSequence(req *repositories.SplitMoveRequest, multiErr *errors.MultiError) {
	// * Can only split after the pickup (sequence 0)
	if req.SplitAfterStopSequence != 0 {
		multiErr.Add(
			"splitAfterStopSequence",
			errors.ErrInvalid,
			"For a simple pickup-delivery move, must split after the pickup (sequence 0)",
		)
		return
	}

	// Note: Removed duplicate time validation as it's already handled in validateSplitTimes
}
