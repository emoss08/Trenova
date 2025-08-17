package shipmentvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"go.uber.org/fx"
)

type ShipmentHoldValidatorParams struct {
	fx.In

	ValidationEngineFactory framework.ValidationEngineFactory
}

type ShipmentHoldValidator struct {
	vef framework.ValidationEngineFactory
}

func NewShipmentHoldValidator(p ShipmentHoldValidatorParams) *ShipmentHoldValidator {
	return &ShipmentHoldValidator{
		vef: p.ValidationEngineFactory,
	}
}

func (v *ShipmentHoldValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	h *shipment.ShipmentHold,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				h.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				v.validateID(h, valCtx, multiErr)

				return nil
			},
		),
	)

	return engine.Validate(ctx)
}

func (v *ShipmentHoldValidator) validateID(
	h *shipment.ShipmentHold,
	valCtx *validator.ValidationContext,
	multiErr *errors.MultiError,
) {
	if valCtx.IsCreate && h.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
