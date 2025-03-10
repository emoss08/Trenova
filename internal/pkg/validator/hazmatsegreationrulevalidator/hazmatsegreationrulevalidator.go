package hazmatsegreationrulevalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
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
	return &Validator{db: p.DB}
}

func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, hsr *hazmatsegregationrule.HazmatSegregationRule) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// * Basic Validation
	hsr.Validate(ctx, multiErr)

	// * Validate Uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, hsr, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// * Validate ID
	v.validateID(hsr, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(ctx context.Context, valCtx *validator.ValidationContext, hsr *hazmatsegregationrule.HazmatSegregationRule, multiErr *errors.MultiError) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(hsr.GetTableName()).
		WithTenant(hsr.OrganizationID, hsr.BusinessUnitID).
		WithModelName("HazmatSegregationRule").
		WithFieldAndTemplate("name", hsr.Name,
			"Hazmat Segregation Rule with name ':value' already exists in the organization.",
			map[string]string{
				"value": hsr.Name,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", hsr.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(hsr *hazmatsegregationrule.HazmatSegregationRule, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && hsr.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
