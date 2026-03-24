package tractorservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*tractor.Tractor]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tractor.Tractor]().
			WithModelName("Tractor").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"code",
				"code",
				"Tractor with this code already exists in your organization",
				func(t *tractor.Tractor) any { return t.Code },
			).
			WithReferenceCheck(
				"equipmentTypeId",
				"equipment_types",
				"Equipment type does not exist or belongs to a different organization",
				func(t *tractor.Tractor) pulid.ID { return t.EquipmentTypeID },
			).
			WithReferenceCheck(
				"equipmentManufacturerId",
				"equipment_manufacturers",
				"Equipment manufacturer does not exist or belongs to a different organization",
				func(t *tractor.Tractor) pulid.ID { return t.EquipmentManufacturerID },
			).
			WithReferenceCheck(
				"primaryWorkerId",
				"workers",
				"Primary worker does not exist or belongs to a different organization",
				func(t *tractor.Tractor) pulid.ID { return t.PrimaryWorkerID },
			).
			WithOptionalReferenceCheck(
				"secondaryWorkerId",
				"workers",
				"Secondary worker does not exist or belongs to a different organization",
				func(t *tractor.Tractor) pulid.ID { return t.SecondaryWorkerID },
			).
			WithOptionalReferenceCheck(
				"fleetCodeId",
				"fleet_codes",
				"Fleet code does not exist or belongs to a different organization",
				func(t *tractor.Tractor) pulid.ID { return t.FleetCodeID },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *tractor.Tractor,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tractor.Tractor,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
