package documenttypevalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billing"
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

func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, dt *billing.DocumentType) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// * Basic Document Type validation
	dt.Validate(ctx, multiErr)

	// * Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, dt, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// * Validate ID
	v.validateID(dt, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(ctx context.Context, valCtx *validator.ValidationContext, dt *billing.DocumentType, multiErr *errors.MultiError) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(dt.GetTableName()).
		WithTenant(dt.OrganizationID, dt.BusinessUnitID).
		WithModelName("DocumentType").
		WithFieldAndTemplate("code", dt.Code,
			"Document type with code ':value' already exists in the organization.",
			map[string]string{
				"value": dt.Code,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", dt.ID.String())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(dt *billing.DocumentType, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && dt.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
