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
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection and validation engine factory.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	MoveRepo                repositories.ShipmentMoveRepository
	ShipmentRepo            repositories.ShipmentRepository
	ShipmentHoldRepo        repositories.ShipmentHoldRepository
	ShipmentHoldValidator   *shipmentvalidator.ShipmentHoldValidator
	ValidationEngineFactory framework.ValidationEngineFactory
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate assignments and other related entities.
type Validator struct {
	db           db.Connection
	moveRepo     repositories.ShipmentMoveRepository
	shipmentRepo repositories.ShipmentRepository
	holdRepo     repositories.ShipmentHoldRepository
	shv          *shipmentvalidator.ShipmentHoldValidator
	vef          framework.ValidationEngineFactory
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
		db:           p.DB,
		moveRepo:     p.MoveRepo,
		shipmentRepo: p.ShipmentRepo,
		holdRepo:     p.ShipmentHoldRepo,
		shv:          p.ShipmentHoldValidator,
		vef:          p.ValidationEngineFactory,
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

	// * I need the shipment status
	shp, err := v.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    move.ShipmentID,
		OrgID: a.OrganizationID,
		BuID:  a.BusinessUnitID,
	})
	if err != nil {
		multiErr.Add(
			"__all__",
			errors.ErrSystemError,
			fmt.Sprintf("failed to get shipment status: %s", err.Error()),
		)

		return
	}

	// * I need the shipment holds
	holds, err := v.holdRepo.GetByShipmentID(ctx, &repositories.GetShipmentHoldByShipmentIDRequest{
		ShipmentID: move.ShipmentID,
		OrgID:      a.OrganizationID,
		BuID:       a.BusinessUnitID,
	})
	if err != nil {
		multiErr.Add(
			"__all__",
			errors.ErrSystemError,
			fmt.Sprintf("failed to get shipment holds: %s", err.Error()),
		)

		return
	}

	// * validate the move can be assigned
	if !v.shv.CanStartTransit(shp.Status, holds.Items) {
		multiErr.Add(
			"tractorId",
			errors.ErrInvalid,
			"Shipment has a blocking hold that prevents dispatch",
		)

		return
	}
}
