package accessorialchargevalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/rotisserie/eris"
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
// It provides methods to validate accessorial charges and other related entities.
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

// Validate validates an accessorial charge.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - acc: The accessorial charge to validate.
//
// Returns:
//   - *errors.MultiError: A MultiError containing validation errors.
func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	acc *accessorialcharge.AccessorialCharge,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				acc.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// * Data integrity validation (uniqueness, references, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				return v.ValidateUniqueness(ctx, valCtx, acc, multiErr)
			},
		),
	)

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of an accessorial charge.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - acc: The accessorial charge to validate.
//   - multiErr: The multi-error to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	acc *accessorialcharge.AccessorialCharge,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(acc.GetTableName()).
		WithTenant(acc.OrganizationID, acc.BusinessUnitID).
		WithModelName("AccessorialCharge").
		WithFieldAndTemplate("code", acc.Code,
			"Accessorial charge with code ':value' already exists in the organization.",
			map[string]string{
				"value": acc.Code,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", acc.ID.String())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}
