package shipmentvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"go.uber.org/fx"
)

type ShipmentHoldValidatorParams struct {
	fx.In

	DB                      db.Connection
	ValidationEngineFactory framework.ValidationEngineFactory
}

type ShipmentHoldValidator struct {
	db  db.Connection
	vef framework.ValidationEngineFactory
}

func NewShipmentHoldValidator(p ShipmentHoldValidatorParams) *ShipmentHoldValidator {
	return &ShipmentHoldValidator{
		db:  p.DB,
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

	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				return v.validateUniqueness(ctx, valCtx, h, multiErr)
			},
		),
	)

	return engine.Validate(ctx)
}

func (v *ShipmentHoldValidator) validateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	h *shipment.ShipmentHold,
	me *errors.MultiError,
) error {
	dba, err := v.db.ReadDB(ctx)
	if err != nil {
		return err
	}

	// Check for the composite unique constraint:
	// UNIQUE(shipment_id, organization_id, business_unit_id, type) WHERE released_at IS NULL
	validator := queryutils.NewCompositeUniquenessValidator(h.GetTableName()).
		WithField("shipment_id", h.ShipmentID).
		WithField("type", h.Type).
		WithTenant(h.OrganizationID, h.BusinessUnitID).
		WithCaseSensitive(true).
		WithErrorField("holdReasonId").
		WithErrorTemplate(
			"An active ':holdType' hold already exists for this shipment. Please release the existing hold before creating a new one.",
			map[string]string{
				"holdType": string(h.Type),
			},
		).
		WithCondition("released_at IS NULL")

	if valCtx.IsCreate {
		validator = validator.ForCreate()
	} else {
		validator = validator.ForUpdate(h.GetID())
	}

	validator.Validate(ctx, dba, me)

	return nil
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
