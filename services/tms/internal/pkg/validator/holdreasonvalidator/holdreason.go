package holdreasonvalidator

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

type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	ValidationEngineFactory framework.ValidationEngineFactory
}

type Validator struct {
	db  db.Connection
	vef framework.ValidationEngineFactory
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:  p.DB,
		vef: p.ValidationEngineFactory,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	hr *shipment.HoldReason,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				hr.Validate(ctx, multiErr)
				v.validateSeverity(hr, multiErr)
				return nil
			},
		),
	)

	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				return v.ValidateUniqueness(ctx, valCtx, hr, multiErr)
			},
		),
	)

	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBusinessRules,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				v.validateID(hr, valCtx, multiErr)
				return nil
			},
		),
	)

	return engine.Validate(ctx)
}

func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	hr *shipment.HoldReason,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.ReadDB(ctx)
	if err != nil {
		return err
	}

	vb := queryutils.NewUniquenessValidator(hr.GetTableName()).
		WithTenant(hr.OrganizationID, hr.BusinessUnitID).
		WithModelName("HoldReason").
		WithFieldAndTemplate("code", hr.Code,
			"Hold reason with code ':value' already exists in the organization.",
			map[string]string{
				"value": hr.Code,
			},
		).
		WithFieldAndTemplate("label", hr.Label,
			"Hold reason with label ':value' already exists in the organization.",
			map[string]string{
				"value": hr.Label,
			},
		)

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", hr.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(
	hr *shipment.HoldReason,
	valCtx *validator.ValidationContext,
	multiErr *errors.MultiError,
) {
	if valCtx.IsCreate && hr.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

func (v *Validator) validateSeverity(
	hr *shipment.HoldReason,
	multiErr *errors.MultiError,
) {
	// * At least one of the blocks must be true if the severity is Blocking
	if hr.DefaultSeverity == shipment.SeverityBlocking {
		if !hr.DefaultBlocksBilling && !hr.DefaultBlocksDelivery && !hr.DefaultBlocksDispatch {
			multiErr.Add(
				"defaultSeverity",
				errors.ErrInvalid,
				"At least one block must be true if the severity is Blocking",
			)
		}
	}
}
