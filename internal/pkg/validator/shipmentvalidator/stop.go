/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type StopValidatorParams struct {
	fx.In

	DB                      db.Connection
	MoveRepo                repositories.ShipmentMoveRepository
	AssignmentRepo          repositories.AssignmentRepository
	Logger                  *logger.Logger
	ValidationEngineFactory framework.ValidationEngineFactory
}

type StopValidator struct {
	db             db.Connection
	moveRepo       repositories.ShipmentMoveRepository
	assignmentRepo repositories.AssignmentRepository
	l              *zerolog.Logger
	vef            framework.ValidationEngineFactory
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
		vef:            p.ValidationEngineFactory,
	}
}

// Validate validates a stop and returns a MultiError if there are any validation errors.
// This is only used for direct stop validation, not when validating stops as part of a move.
// Stop validations as part of a move are done in MoveValidator.validateStopTimes.
func (v *StopValidator) Validate(
	ctx context.Context,
	s *shipment.Stop,
	idx int,
	multiErr *errors.MultiError,
) {
	engine := v.vef.CreateEngine().
		ForField("stops").
		AtIndex(idx).
		WithParent(multiErr)

	// Basic stop validation
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				s.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// Time validation
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBusinessRules,
			framework.ValidationPriorityHigh,
			func(_ context.Context, multiErr *errors.MultiError) error {
				v.validateTimes(s, multiErr)
				return nil
			},
		),
	)

	// Assignment validation
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBusinessRules,
			framework.ValidationPriorityMedium,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				if s.ActualArrival != nil || s.ActualDeparture != nil {
					v.validateAssignment(ctx, s, multiErr)
				}
				return nil
			},
		),
	)

	// Execute validation - intentionally discard return value as errors are added to parent
	_ = engine.Validate(ctx)
}

// validateTimes validates the planned and actual arrival/departure times for a stop.
func (v *StopValidator) validateTimes(s *shipment.Stop, multiErr *errors.MultiError) {
	// Validate planned arrival/departure times for the stop
	if s.PlannedArrival > s.PlannedDeparture {
		multiErr.Add(
			"plannedArrival",
			errors.ErrInvalid,
			"Planned arrival must be before planned departure",
		)
	}

	// Validate actual arrival/departure times if both are set
	if s.ActualArrival != nil && s.ActualDeparture != nil {
		if *s.ActualArrival > *s.ActualDeparture {
			multiErr.Add(
				"actualArrival",
				errors.ErrInvalid,
				"Actual arrival must be before actual departure",
			)
		}
	}

	// Validate that actual arrival time cannot be in the future
	if s.ActualArrival != nil {
		currentTime := timeutils.NowUnix()
		if *s.ActualArrival > currentTime {
			multiErr.Add(
				"actualArrival",
				errors.ErrInvalid,
				"Actual arrival time cannot be in the future",
			)
		}
	}

	// Validate that actual departure time cannot be in the future
	if s.ActualDeparture != nil {
		currentTime := timeutils.NowUnix()
		if *s.ActualDeparture > currentTime {
			multiErr.Add(
				"actualDeparture",
				errors.ErrInvalid,
				"Actual departure time cannot be in the future",
			)
		}
	}
}

// validateAssignment checks if the move has an assignment when actual times are set.
func (v *StopValidator) validateAssignment(
	ctx context.Context,
	s *shipment.Stop,
	multiErr *errors.MultiError,
) {
	move, err := v.moveRepo.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            s.ShipmentMoveID,
		OrgID:             s.OrganizationID,
		BuID:              s.BusinessUnitID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		v.l.Error().Err(err).Interface("stop", s).Msgf("failed to get move for stop %s", s.ID)
		multiErr.Add(
			"shipmentMoveID",
			errors.ErrInvalid,
			fmt.Sprintf("Shipment move (%s) not found: %s", s.ShipmentMoveID, err),
		)
		return
	}

	if move.Assignment == nil && (s.ActualArrival != nil || s.ActualDeparture != nil) {
		if s.ActualArrival != nil {
			multiErr.Add(
				"actualArrival",
				errors.ErrInvalid,
				"Actual arrival and departure times cannot be set on a move with no assignment",
			)
		}
		if s.ActualDeparture != nil {
			multiErr.Add(
				"actualDeparture",
				errors.ErrInvalid,
				"Actual arrival and departure times cannot be set on a move with no assignment",
			)
		}
	}
}
