package consolidationvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
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
// It provides methods to validate consolidation groups and other related entities.
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

// Validate validates a consolidation group.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - cg: The consolidation group to validate.
//
// Returns:
//   - *errors.MultiError: A MultiError containing validation errors.
func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	cg *consolidation.ConsolidationGroup,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	// engine.AddRule(
	// 	framework.NewValidationRule(
	// 		framework.ValidationStageBasic,
	// 		framework.ValidationPriorityHigh,
	// 		func(c context.Context, multiErr *errors.MultiError) error {
	// 			cg.Validate(c, multiErr)
	// 			return nil
	// 		},
	// 	),
	// )

	// * Data integrity validation (uniqueness, references, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(c context.Context, multiErr *errors.MultiError) error {
				return v.ValidateUniqueness(c, valCtx, cg, multiErr)
			},
		),
	)

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of a consolidation group.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - cg: The consolidation group to validate.
//   - multiErr: The multi error to add errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	cg *consolidation.ConsolidationGroup,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.ReadDB(ctx)
	if err != nil {
		return err
	}

	vb := queryutils.NewUniquenessValidator(cg.GetTableName()).
		WithTenant(cg.OrganizationID, cg.BusinessUnitID).
		WithModelName("ConsolidationGroup").
		WithFieldAndTemplate("consolidation_number", cg.ConsolidationNumber,
			"Consolidation group with consolidation number ':value' already exists in the organization.",
			map[string]string{
				"value": cg.ConsolidationNumber,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).WithPrimaryKey("id", cg.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}
