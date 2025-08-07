/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package assignmentvalidator

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

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection and validation engine factory.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	MoveRepo                repositories.ShipmentMoveRepository
	ValidationEngineFactory framework.ValidationEngineFactory
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate assignments and other related entities.
type Validator struct {
	db       db.Connection
	moveRepo repositories.ShipmentMoveRepository
	vef      framework.ValidationEngineFactory
}

// NewValidator initializes a new Validator with the provided dependencies.
//
// Parameters:
//   - p: ValidatorParams containing dependencies.
//
// Returns:
//   - *Validator: A new Validator instance.
func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:       p.DB,
		moveRepo: p.MoveRepo,
		vef:      p.ValidationEngineFactory,
	}
}

// Validate validates an assignment.
//
// Parameters:
//   - ctx: The context of the request.
//   - a: The assignment to validate.
//
// Returns:
//   - *errors.MultiError: A MultiError containing validation errors.
func (v *Validator) Validate(ctx context.Context, a *shipment.Assignment) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Business rules validation (domain-specific rules)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				v.validateAssignmentCriteria(ctx, a, multiErr)
				return nil
			},
		),
	)

	return engine.Validate(ctx)
}

// validateAssignmentCriteria validates the assignment criteria.
//
// Parameters:
//   - ctx: The context of the request.
//   - a: The assignment to validate.
//   - multiErr: The multi-error to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) validateAssignmentCriteria(
	ctx context.Context,
	a *shipment.Assignment,
	multiErr *errors.MultiError,
) {
	move, err := v.moveRepo.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID: a.ShipmentMoveID,
		OrgID:  a.OrganizationID,
		BuID:   a.BusinessUnitID,
	})
	if err != nil {
		multiErr.Add(
			"move",
			errors.ErrSystemError,
			fmt.Sprintf("failed to get move: %s", err.Error()),
		)
		return
	}

	if !assignableMoveStatuses[move.Status] {
		multiErr.Add(
			"__all__",
			errors.ErrInvalid,
			fmt.Sprintf("Cannot assign to a move that is in the `%s` status", move.Status),
		)

		return
	}
}
