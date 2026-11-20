package costingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

type Validator struct {
	controlValidator  tenantedvalidation.EntityReporter[*costingcontrol.CostingControl]
	categoryValidator tenantedvalidation.EntityReporter[*costingcontrol.CostCategory]
}

func NewValidator() *Validator {
	return &Validator{
		controlValidator: tenantedvalidation.NewBuilder[*costingcontrol.CostingControl]().
			WithModelName("CostingControl").
			WithCustomRule(createGLActualsRule()).
			Build(),
		categoryValidator: tenantedvalidation.NewBuilder[*costingcontrol.CostCategory]().
			WithModelName("CostCategory").
			Build(),
	}
}

func createGLActualsRule() validationframework.Rule[*costingcontrol.CostingControl] {
	return validationframework.NewRule[*costingcontrol.CostingControl]("gl_actuals").
		OnUpdate().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *costingcontrol.CostingControl,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.GLActualsEnabled {
				return nil
			}

			if entity.GLRollingMonths < 1 || entity.GLRollingMonths > 12 {
				multiErr.Add(
					"glRollingMonths",
					errortypes.ErrInvalid,
					"GL rolling months must be between 1 and 12",
				)
			}

			if entity.PlannedMonthlyMiles != nil && *entity.PlannedMonthlyMiles < 1 {
				multiErr.Add(
					"plannedMonthlyMiles",
					errortypes.ErrInvalid,
					"Planned monthly miles must be greater than 0",
				)
			}

			return nil
		})
}

func (v *Validator) ValidateSave(ctx context.Context, plan *SavePlan) error {
	multiErr := errortypes.NewMultiError()

	if plan.ControlChanged {
		addReport(
			multiErr.WithPrefix("control"),
			v.controlValidator.ReportUpdate(ctx, plan.Control),
		)
		if multiErr.Internal() != nil {
			return multiErr.Err()
		}
	}

	seen := make(map[pulid.ID]struct{}, len(plan.Categories))
	for i, planned := range plan.Categories {
		scoped := multiErr.WithIndex("categories", i)
		if planned.Category == nil {
			scoped.Add("id", errortypes.ErrNotFound, "Cost category not found")
			continue
		}
		if _, duplicate := seen[planned.Category.ID]; duplicate {
			scoped.Add(
				"id",
				errortypes.ErrDuplicate,
				"A cost category can be changed once per save",
			)
			continue
		}
		seen[planned.Category.ID] = struct{}{}

		addReport(scoped, v.categoryValidator.ReportUpdate(ctx, planned.Category))
		if multiErr.Internal() != nil {
			return multiErr.Err()
		}

		if needsGLAccount(plan.Control, planned.Category, len(planned.GLAccountIDs)) {
			scoped.Add(
				"glAccountIds",
				errortypes.ErrRequired,
				"Map at least one GL account to use GL actuals for this category",
			)
		}
	}

	for _, category := range plan.Unchanged {
		if needsGLAccount(plan.Control, category, len(category.GLAccounts)) {
			multiErr.Add(
				"control.glActualsEnabled",
				errortypes.ErrInvalid,
				"{0} reads its rate from GL actuals and has no GL account mapped",
				category.Name,
			)
		}
	}

	return multiErr.Err()
}

func needsGLAccount(
	control *costingcontrol.CostingControl,
	category *costingcontrol.CostCategory,
	glAccountCount int,
) bool {
	return control.GLActualsEnabled &&
		category.IsActive &&
		category.RateSource == costingcontrol.RateSourceGLActual &&
		glAccountCount == 0
}

func addReport(dst, report *errortypes.MultiError) {
	if report == nil {
		return
	}
	for _, err := range report.Errors {
		dst.AddError(err)
	}
	dst.AddInternal(report.Internal())
}
