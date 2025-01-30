package shipmentvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"go.uber.org/fx"
)

type StopValidatorParams struct {
	fx.In

	DB db.Connection
}

type StopValidator struct {
	db db.Connection
}

func NewStopValidator(p StopValidatorParams) *StopValidator {
	return &StopValidator{
		db: p.DB,
	}
}

func (v *StopValidator) Validate(ctx context.Context, valCtx *validator.ValidationContext, s *shipment.Stop, multiErr *errors.MultiError, idx int) {
	stopMultiErr := multiErr.WithIndex("stops", idx)

	s.Validate(ctx, stopMultiErr)

	if valCtx.IsCreate {
		v.validateID(s, stopMultiErr)
	}

	v.validateTimes(stopMultiErr, s)
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
