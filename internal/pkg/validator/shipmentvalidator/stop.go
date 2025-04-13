package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type StopValidatorParams struct {
	fx.In

	DB             db.Connection
	MoveRepo       repositories.ShipmentMoveRepository
	AssignmentRepo repositories.AssignmentRepository
	Logger         *logger.Logger
}

type StopValidator struct {
	db             db.Connection
	moveRepo       repositories.ShipmentMoveRepository
	assignmentRepo repositories.AssignmentRepository
	l              *zerolog.Logger
	multiErr       *errors.MultiError
}

func NewStopValidator(p StopValidatorParams) *StopValidator {
	log := p.Logger.With().
		Str("validator", "stop").
		Logger()

	return &StopValidator{
		db:             p.DB,
		moveRepo:       p.MoveRepo,
		assignmentRepo: p.AssignmentRepo,
		l:              &log,
	}
}

type StopValidatorOption func(*StopValidator)

func WithIndexedMultiError(multiErr *errors.MultiError, idx int) StopValidatorOption {
	return func(v *StopValidator) {
		v.multiErr = multiErr.WithIndex("stops", idx)
	}
}

func (v *StopValidator) Validate(ctx context.Context, s *shipment.Stop, opts ...StopValidatorOption) *errors.MultiError {
	if v.multiErr == nil {
		v.multiErr = errors.NewMultiError()
	}

	for _, opt := range opts {
		opt(v)
	}

	s.Validate(ctx, v.multiErr)

	v.validateTimes(ctx, v.multiErr, s)

	if v.multiErr.HasErrors() {
		return v.multiErr
	}

	return nil
}

func (v *StopValidator) validateTimes(ctx context.Context, multiErr *errors.MultiError, s *shipment.Stop) {
	log := v.l.With().
		Str("operation", "validateTimes").
		Str("stopID", s.ID.String()).
		Logger()

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

	// * If there is movement has no assignment, the stop cannot have actual times
	move, err := v.moveRepo.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            s.ShipmentMoveID,
		OrgID:             s.OrganizationID,
		BuID:              s.BusinessUnitID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		log.Error().Err(err).Interface("stop", s).Msgf("failed to get move for stop %s", s.ID)
		multiErr.Add("shipmentMoveID", errors.ErrInvalid, fmt.Sprintf("Shipment move (%s) not found: %s", s.ShipmentMoveID, err))
		return
	}

	if move.Assignment == nil {
		if s.ActualArrival != nil || s.ActualDeparture != nil {
			multiErr.Add("actualArrival", errors.ErrInvalid, "Actual arrival cannot be set on a move with no assignment")
		}
	}
}
