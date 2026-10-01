package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveWorkerCredentialTypeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.WorkerCredentialService.GetType(
				ctx,
				&repositories.GetCredentialTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, workerCredentialTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.WorkerCredentialService.TypeSelectOptions(
		ctx,
		&repositories.WorkerCredentialTypeSelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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
			Description: stringutils.Ptr(entity.Description),
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
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.WorkerTrainingService.GetCourse(
				ctx,
				&repositories.GetTrainingCourseByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, trainingCourseSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.WorkerTrainingService.CourseSelectOptions(
		ctx,
		&repositories.TrainingCourseSelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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
		"validityMonths":         int32Value(entity.ValidityMonths),
		"passingScore":           nil,
	}
	if entity.PassingScore.Valid {
		meta["passingScore"] = entity.PassingScore.Decimal.String()
	}

	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
			Meta:        meta,
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolvePerformanceReviewTemplateSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.PerformanceReviewService.GetTemplate(
				ctx,
				&repositories.GetReviewTemplateByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, performanceReviewTemplateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.PerformanceReviewService.TemplateSelectOptions(
		ctx,
		&repositories.ReviewTemplateSelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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
			Description: stringutils.Ptr(entity.Description),
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
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.PtoPolicyService.Get(ctx, &repositories.GetPTOPolicyByIDRequest{
				ID:           id,
				TenantInfo:   req.TenantInfo,
				IncludeRules: true,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, ptoPolicySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.PtoPolicyService.SelectOptions(
		ctx,
		&repositories.PTOPolicySelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		ptoPolicySelectOptionItem,
	)
}

func ptoPolicySelectOptionItem(entity *worker.PTOPolicy) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
			Meta: map[string]any{
				"code":             entity.Code,
				"isDefault":        entity.IsDefault,
				"yearBasis":        string(entity.YearBasis),
				"requiresApproval": entity.RequiresApproval,
				"ptoTypes":         ptoPolicyTrackedTypes(entity),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolveBenefitPlanSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.BenefitsService.GetPlan(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, benefitPlanSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.BenefitsService.PlanSelectOptions(
		ctx,
		&repositories.BenefitPlanSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			PlanYear:           selectOptionInt16Filter(req.Filters, "planYear"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		benefitPlanSelectOptionItem,
	)
}

func benefitPlanSelectOptionItem(entity *driverpay.BenefitPlan) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
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

// ptoPolicyTrackedTypes is what the assignment form turns into one opening
// balance row per type, so a policy with no rules has to report an empty list
// rather than a null the form would have to guard.
func ptoPolicyTrackedTypes(entity *worker.PTOPolicy) []string {
	types := make([]string, 0, len(entity.Rules))
	for _, rule := range entity.Rules {
		types = append(types, string(rule.PTOType))
	}

	return types
}
