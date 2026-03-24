package trailerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
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
	validator *validationframework.TenantedValidator[*trailer.Trailer]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*trailer.Trailer]().
			WithModelName("Trailer").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"code",
				"code",
				"Trailer with this code already exists in your organization",
				func(t *trailer.Trailer) any { return t.Code },
			).
			WithReferenceCheck(
				"equipmentTypeId",
				"equipment_types",
				"Equipment type does not exist or belongs to a different organization",
				func(t *trailer.Trailer) pulid.ID { return t.EquipmentTypeID },
			).
			WithReferenceCheck(
				"equipmentManufacturerId",
				"equipment_manufacturers",
				"Equipment manufacturer does not exist or belongs to a different organization",
				func(t *trailer.Trailer) pulid.ID { return t.EquipmentManufacturerID },
			).
			WithOptionalReferenceCheck(
				"fleetCodeId",
				"fleet_codes",
				"Fleet code does not exist or belongs to a different organization",
				func(t *trailer.Trailer) pulid.ID { return t.FleetCodeID },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *trailer.Trailer,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *trailer.Trailer,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
