package servicetypevalidator

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain/servicetype"
	"github.com/trenova-app/transport/internal/core/ports/db"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/queryutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
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
	st *servicetype.ServiceType,
) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Basic Service Type validation
	st.Validate(ctx, multiErr)

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, st, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(st, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(ctx context.Context, valCtx *validator.ValidationContext, st *servicetype.ServiceType, multiErr *errors.MultiError) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(st.GetTableName()).
		WithTenant(st.OrganizationID, st.BusinessUnitID).
		WithModelName("ShipmentType").
		WithFieldAndTemplate("code", st.Code,
			"Service Type with code ':value' already exists in the organization.",
			map[string]string{
				"value": st.Code,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", st.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(st *servicetype.ServiceType, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && st.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
