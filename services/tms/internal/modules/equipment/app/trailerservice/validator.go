package trailerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*trailer.Trailer]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*trailer.Trailer]().
			WithModelName("Trailer").
			WithIntegrityChecker(p.Integrity).
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *trailer.Trailer,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
