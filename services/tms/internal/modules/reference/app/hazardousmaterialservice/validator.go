package hazardousmaterialservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*hazardousmaterial.HazardousMaterial]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*hazardousmaterial.HazardousMaterial]().
			WithModelName("HazardousMaterial").
			WithIntegrityChecker(p.Integrity).
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
