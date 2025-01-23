package equipmenttypevalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
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
	et *equipmenttype.EquipmentType,
) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Basic Equipment Type validation
	et.Validate(ctx, multiErr)

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, et, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(et, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	et *equipmenttype.EquipmentType,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(et.GetTableName()).
		WithTenant(et.OrganizationID, et.BusinessUnitID).
		WithModelName("EquipmentType").
		WithFieldAndTemplate("code", et.Code,
			"Equipment Type with code ':value' already exists in the organization.",
			map[string]string{
				"value": et.Code,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", et.ID.String())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(et *equipmenttype.EquipmentType, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && et.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
