package shipmentvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"go.uber.org/fx"
)

type StopValidatorParams struct {
	fx.In

	DB       db.Connection
	MoveRepo repositories.ShipmentMoveRepository
}

type StopValidator struct {
	db       db.Connection
	moveRepo repositories.ShipmentMoveRepository
	multiErr *errors.MultiError
}

func NewStopValidator(p StopValidatorParams) *StopValidator {
	return &StopValidator{
		db:       p.DB,
		moveRepo: p.MoveRepo,
	}
}

type StopValidatorOption func(*StopValidator)

func WithIndexedMultiError(multiErr *errors.MultiError, idx int) StopValidatorOption {
	return func(v *StopValidator) {
		v.multiErr = multiErr.WithIndex("stops", idx)
	}
}

func (v *StopValidator) Validate(ctx context.Context, valCtx *validator.ValidationContext, s *shipment.Stop, opts ...StopValidatorOption) *errors.MultiError {
	if v.multiErr == nil {
		v.multiErr = errors.NewMultiError()
	}

	for _, opt := range opts {
		opt(v)
	}

	s.Validate(ctx, v.multiErr)

	if valCtx.IsCreate {
		v.validateID(s, v.multiErr)
	}

	v.validateTimes(v.multiErr, s)

	if v.multiErr.HasErrors() {
		return v.multiErr
	}

	return nil
}

func (v *StopValidator) validateID(s *shipment.Stop, multiErr *errors.MultiError) {
	if s.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

func (v *StopValidator) validateTimes(multiErr *errors.MultiError, s *shipment.Stop) {
	if s.PlannedArrival > s.PlannedDeparture {
		multiErr.Add("plannedArrival", errors.ErrInvalid, "Planned arrival must be before planned departure")
	}

	if s.PlannedDeparture < s.PlannedArrival {
		multiErr.Add("plannedDeparture", errors.ErrInvalid, "Planned departure must be after planned arrival")
	}

	if s.ActualArrival != nil && s.ActualDeparture != nil && *s.ActualArrival > *s.ActualDeparture {
		multiErr.Add("actualArrival", errors.ErrInvalid, "Actual arrival must be before actual departure")
	}

	if s.ActualArrival != nil && s.ActualDeparture != nil && *s.ActualDeparture < *s.ActualArrival {
		multiErr.Add("actualDeparture", errors.ErrInvalid, "Actual departure must be after actual arrival")
	}
}

// validation for stop sequence
// ensures that if a user is updating a stop.
// 1. the previous moves stops are sequenced // Example: if the previous move is not completed, then you can't update this stops times
// 2. if the previous stop is not completed, the you can't update this stop's times.

// Implementation:
// 1. get all moves for the shipment
// 2. loop through the moves and check if the previous stop is completed (We can just subtract 1 from the current stop sequence number)
// 3. if the previous stop is not completed, then we can't update this stop's times.
func (v *StopValidator) validateStopSequence(multiErr *errors.MultiError, s *shipment.Stop) {
	// Get the move by stop id
	
}