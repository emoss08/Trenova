package fleetcodevalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB db.Connection
}

type Validator struct {
	db db.Connection
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db: p.DB,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	fc *fleetcode.FleetCode,
) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Basic fleet code validation
	fc.Validate(ctx, multiErr)

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, fc, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(fc, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	fc *fleetcode.FleetCode,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(fc.GetTableName()).
		WithTenant(fc.OrganizationID, fc.BusinessUnitID).
		WithModelName("FleetCode").
		WithFieldAndTemplate("name", fc.Name,
			"Fleet code with name ':value' already exists in the organization.",
			map[string]string{
				"value": fc.Name,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", fc.ID.String())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(fc *fleetcode.FleetCode, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && fc.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
