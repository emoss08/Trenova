package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *Resolver) resolveWorkerCredentialTypeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.workerCredentialService.GetType(
				ctx,
				&repositories.GetCredentialTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, workerCredentialTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.workerCredentialService.TypeSelectOptions(
		ctx,
		&repositories.WorkerCredentialTypeSelectOptionsRequest{SelectQueryRequest: req.selectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		workerCredentialTypeSelectOptionItem,
	)
}

func workerCredentialTypeSelectOptionItem(
	entity *worker.WorkerCredentialType,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringPtr(entity.Description),
			Meta: map[string]any{
				"code":             entity.Code,
				"category":         string(entity.Category),
				"isRequired":       entity.IsRequired,
				"requiresNumber":   entity.RequiresNumber,
				"requiresDocument": entity.RequiresDocument,
				"profileField":     string(entity.ProfileField),
				"validityMonths":   int32Value(entity.ValidityMonths),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolveTrainingCourseSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.workerTrainingService.GetCourse(
				ctx,
				&repositories.GetTrainingCourseByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, trainingCourseSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.workerTrainingService.CourseSelectOptions(
		ctx,
		&repositories.TrainingCourseSelectOptionsRequest{SelectQueryRequest: req.selectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		trainingCourseSelectOptionItem,
	)
}

func trainingCourseSelectOptionItem(entity *worker.TrainingCourse) selectOptionConnectionItem {
	meta := map[string]any{
		"code":                   entity.Code,
		"category":               string(entity.Category),
		"delivery":               string(entity.Delivery),
		"durationMinutes":        entity.DurationMinutes,
		"dueDaysAfterAssignment": entity.DueDaysAfterAssignment,
		"passingScore":           nil,
	}
	if entity.PassingScore.Valid {
		meta["passingScore"] = entity.PassingScore.Decimal.String()
	}

	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringPtr(entity.Description),
			Meta:        meta,
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolvePerformanceReviewTemplateSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.performanceReviewService.GetTemplate(
				ctx,
				&repositories.GetReviewTemplateByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, performanceReviewTemplateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.performanceReviewService.TemplateSelectOptions(
		ctx,
		&repositories.ReviewTemplateSelectOptionsRequest{SelectQueryRequest: req.selectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		performanceReviewTemplateSelectOptionItem,
	)
}

func performanceReviewTemplateSelectOptionItem(
	entity *worker.PerformanceReviewTemplate,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringPtr(entity.Description),
			Meta: map[string]any{
				"code":          entity.Code,
				"isDefault":     entity.IsDefault,
				"itemCount":     len(entity.Items),
				"cadenceMonths": int32Value(entity.CadenceMonths),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolvePTOPolicySelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ptoPolicyService.Get(ctx, &repositories.GetPTOPolicyByIDRequest{
				ID:         id,
				TenantInfo: req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, ptoPolicySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ptoPolicyService.SelectOptions(
		ctx,
		&repositories.PTOPolicySelectOptionsRequest{SelectQueryRequest: req.selectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ptoPolicySelectOptionItem,
	)
}

func ptoPolicySelectOptionItem(entity *worker.PTOPolicy) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringPtr(entity.Description),
			Meta: map[string]any{
				"code":             entity.Code,
				"isDefault":        entity.IsDefault,
				"yearBasis":        string(entity.YearBasis),
				"requiresApproval": entity.RequiresApproval,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolveBenefitPlanSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.benefitsService.GetPlan(ctx, req.tenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, benefitPlanSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.benefitsService.PlanSelectOptions(
		ctx,
		&repositories.BenefitPlanSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			PlanYear:           selectOptionInt16Filter(req.filters, "planYear"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		benefitPlanSelectOptionItem,
	)
}

func benefitPlanSelectOptionItem(entity *driverpay.BenefitPlan) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringPtr(entity.Description),
			Meta: map[string]any{
				"code":              entity.Code,
				"planType":          string(entity.PlanType),
				"planYear":          entity.PlanYear,
				"carrier":           entity.Carrier,
				"employeeCostMinor": entity.EmployeeCostMinor,
				"employerCostMinor": entity.EmployerCostMinor,
				"currencyCode":      entity.CurrencyCode,
				"waitingPeriodDays": entity.WaitingPeriodDays,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func int32Value(value *int32) any {
	if value == nil {
		return nil
	}
	return *value
}
