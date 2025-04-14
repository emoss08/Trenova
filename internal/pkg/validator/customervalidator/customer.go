package customervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
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
// It provides methods to validate customers and other related entities.
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

// Validate validates a customer.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - cus: The customer to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, cus *customer.Customer) *errors.MultiError {
	engine := v.vef.CreateEngine()

	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			cus.Validate(ctx, multiErr)
			return nil
		}))

	// Validate uniqueness
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageDataIntegrity, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			return v.ValidateUniqueness(ctx, valCtx, cus, multiErr)
		}))

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of a customer.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - cus: The customer to validate.
//   - multiErr: The MultiError to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) ValidateUniqueness(
	ctx context.Context, valCtx *validator.ValidationContext, cus *customer.Customer, multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(cus.GetTableName()).
		WithTenant(cus.OrganizationID, cus.BusinessUnitID).
		WithModelName("Customer").
		WithFieldAndTemplate("code", cus.Code,
			"Customer with code ':value' already exists in the organization.",
			map[string]string{
				"value": cus.Code,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", cus.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}
