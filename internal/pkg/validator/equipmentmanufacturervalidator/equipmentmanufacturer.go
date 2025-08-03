/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package equipmentmanufacturervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection and validation engine factory.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	ValidationEngineFactory framework.ValidationEngineFactory
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate equipment manufacturers and other related entities.
type Validator struct {
	db  db.Connection
	vef framework.ValidationEngineFactory
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
		db:  p.DB,
		vef: p.ValidationEngineFactory,
	}
}

// Validate validates an equipment manufacturer.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - em: The equipment manufacturer to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	em *equipmentmanufacturer.EquipmentManufacturer,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				em.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// * Data Integrity Validation (uniqueness, references, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				return v.ValidateUniqueness(ctx, valCtx, em, multiErr)
			},
		),
	)

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of an equipment manufacturer.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - em: The equipment manufacturer to validate.
//   - multiErr: The MultiError to add validation errors to.
func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	em *equipmentmanufacturer.EquipmentManufacturer,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(em.GetTableName()).
		WithTenant(em.OrganizationID, em.BusinessUnitID).
		WithModelName("EquipmentManufacturer").
		WithFieldAndTemplate("name", em.Name,
			"Equipment Manufacturer with name ':value' already exists in the organization.",
			map[string]string{
				"value": em.Name,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		log.Debug().Msg("We're hitting this currently.")
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", em.ID.String())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}
