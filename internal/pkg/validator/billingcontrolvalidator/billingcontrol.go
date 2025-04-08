package billingcontrolvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
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

func (v *Validator) Validate(ctx context.Context, _ *validator.ValidationContext, bc *billing.BillingControl) *errors.MultiError {
	multiErr := errors.NewMultiError()

	bc.Validate(ctx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
