package locationservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*location.Location]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*location.Location]().
			WithModelName("Location").
			WithIntegrityChecker(p.Integrity).
			WithUniqueField(
				"code",
				"code",
				"Location with this code already exists in your organization",
				func(l *location.Location) any { return l.Code },
			).
			WithReferenceCheck(
				"stateId",
				"us_states",
				"State does not exist",
				func(l *location.Location) pulid.ID { return l.StateID },
			).
			WithReferenceCheck(
				"locationCategoryId",
				"location_categories",
				"Location category does not exist",
				func(l *location.Location) pulid.ID { return l.LocationCategoryID },
			).
			WithCustomRule(codeRequiredOnUpdateRule()).
			Build(),
	}
}

func codeRequiredOnUpdateRule() validationframework.Rule[*location.Location] {
	return validationframework.NewRule[*location.Location]("code_required_on_update").
		OnUpdate().
		WithStage(validationframework.StageBasic).
		WithValidation(func(
			_ context.Context,
			entity *location.Location,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if strings.TrimSpace(entity.Code) == "" && !multiErr.HasErrorAt("code") {
				multiErr.Add("code", errortypes.ErrRequired, "Code is required")
			}

			return nil
		})
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *location.Location,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *location.Location,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
