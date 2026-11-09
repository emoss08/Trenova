package locationcategoryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*locationcategory.LocationCategory]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*locationcategory.LocationCategory]().
			WithModelName("Location Category").
			WithIntegrityChecker(
				p.Integrity,
			).
			WithUniqueField(
				"name",
				"name",
				"Location category with this name already exists in your organization",
				func(e *locationcategory.LocationCategory) any { return e.Name },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *locationcategory.LocationCategory,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *locationcategory.LocationCategory,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
