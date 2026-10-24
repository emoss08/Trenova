package equipmenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*equipmenttype.EquipmentType]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(p.DB, equipmenttypedb.EquipmentTypesSchema())

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*equipmenttype.EquipmentType]().
			WithModelName("Equipment Type").
			WithUniquenessChecker(checker).
			WithUniqueField(
				"code",
				"code",
				"Equipment type with this code already exists in your organization",
				func(e *equipmenttype.EquipmentType) any { return e.Code },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *equipmenttype.EquipmentType,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *equipmenttype.EquipmentType,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
