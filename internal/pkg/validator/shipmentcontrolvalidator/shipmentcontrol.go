package shipmentcontrolvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
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

func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, sc *shipment.ShipmentControl) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Basic Shipment Control validation
	sc.Validate(ctx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
