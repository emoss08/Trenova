package locationvalidator

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain/location"
	"github.com/trenova-app/transport/internal/core/ports/db"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/queryutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"go.uber.org/fx"
)

type LocationCategoryValidatorParams struct {
	fx.In

	DB db.Connection
}

type LocationCategoryValidator struct {
	db db.Connection
}

func NewLocationCategoryValidator(p LocationCategoryValidatorParams) *LocationCategoryValidator {
	return &LocationCategoryValidator{
		db: p.DB,
	}
}

func (v *LocationCategoryValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	lc *location.LocationCategory,
) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Basic Location Category validation
	lc.Validate(ctx, multiErr)

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, lc, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(lc, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *LocationCategoryValidator) ValidateUniqueness(ctx context.Context, valCtx *validator.ValidationContext, lc *location.LocationCategory, multiErr *errors.MultiError) error {
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

func (v *LocationCategoryValidator) validateID(lc *location.LocationCategory, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && lc.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
