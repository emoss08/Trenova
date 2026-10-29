package locationcategoryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*locationcategory.LocationCategory]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*locationcategory.LocationCategory]().
			WithModelName("Location Category").
			WithUniquenessChecker(
				keelcheck.NewChecker(p.DB, locationcategorydb.LocationCategoriesSchema()),
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
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *locationcategory.LocationCategory,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
