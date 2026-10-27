package hazardousmaterialservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial/hazardousmaterialdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*hazardousmaterial.HazardousMaterial]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(
		p.DB,
		hazardousmaterialdb.HazardousMaterialsSchema(),
	)

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*hazardousmaterial.HazardousMaterial]().
			WithModelName("HazardousMaterial").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithUniqueField(
				"name",
				"name",
				"Hazardous material with this name already exists in your organization",
				func(hm *hazardousmaterial.HazardousMaterial) any { return hm.Name },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
