package locationvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

// LocationCategoryValidatorParams defines the dependencies required for initializing the LocationCategoryValidator.
// This includes the database connection and validation engine factory.
type LocationCategoryValidatorParams struct {
	fx.In

	DB                      db.Connection
	ValidationEngineFactory framework.ValidationEngineFactory
}

// LocationCategoryValidator is a validator for location categories.
// It validates location categories, and other related entities.
type LocationCategoryValidator struct {
	db  db.Connection
	vef framework.ValidationEngineFactory
}

// NewLocationCategoryValidator initializes a new LocationCategoryValidator with the provided dependencies.
//
// Parameters:
//   - p: LocationCategoryValidatorParams containing dependencies.
//
// Returns:
//   - *LocationCategoryValidator: A new LocationCategoryValidator instance.
func NewLocationCategoryValidator(p LocationCategoryValidatorParams) *LocationCategoryValidator {
	return &LocationCategoryValidator{
		db:  p.DB,
		vef: p.ValidationEngineFactory,
	}
}

// Validate validates a location category.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - lc: The location category to validate.
//
// Returns:
//   - *errors.MultiError: A MultiError containing validation errors.
func (v *LocationCategoryValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	lc *location.LocationCategory,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				lc.Validate(ctx, multiErr)
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
				return v.ValidateUniqueness(ctx, valCtx, lc, multiErr)
			},
		),
	)

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of a location category.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - lc: The location category to validate.
//   - multiErr: The multi-error to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *LocationCategoryValidator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	lc *location.LocationCategory,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(lc.GetTableName()).
		WithTenant(lc.OrganizationID, lc.BusinessUnitID).
		WithModelName("LocationCategory").
		WithFieldAndTemplate("name", lc.Name,
			"Location Category with name ':value' already exists in the organization.",
			map[string]string{
				"value": lc.Name,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", lc.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}
