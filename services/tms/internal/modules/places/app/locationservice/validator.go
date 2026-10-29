package locationservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*location.Location]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, locationdb.LocationsSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*location.Location]().
			WithModelName("Location").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithUniqueField(
				"code",
				"code",
				"Location with this code already exists in your organization",
				func(l *location.Location) any { return l.Code },
			).
			WithCustomReferenceCheck(
				"stateId",
				"State does not exist",
				func(l *location.Location) pulid.ID { return l.StateID },
				keelcheck.ReferenceCheck(p.DB, usstatedb.UsStatesSchema()),
			).
			WithCustomReferenceCheck(
				"locationCategoryId",
				"Location category does not exist",
				func(l *location.Location) pulid.ID { return l.LocationCategoryID },
				keelcheck.ReferenceCheck(
					p.DB,
					locationcategorydb.LocationCategoriesSchema(),
				),
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *location.Location,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *location.Location,
) *errortypes.MultiError {
	multiErr := v.validator.ValidateUpdate(ctx, entity)
	if multiErr == nil {
		multiErr = errortypes.NewMultiError()
	}
	if strings.TrimSpace(entity.Code) == "" {
		multiErr.Add("code", errortypes.ErrRequired, "Code is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
