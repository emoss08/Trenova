package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolveWorkerCredentialTypeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.WorkerCredentialService.GetType(
				ctx,
				&workerports.GetCredentialTypeByIDRequest{
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
		&workerports.WorkerCredentialTypeSelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		workerCredentialTypeSelectOptionItem,
	)
}

func workerCredentialTypeSelectOptionItem(
	entity *workercredential.WorkerCredentialType,
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

func (r *Deps) resolveTrainingCourseSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.WorkerTrainingService.GetCourse(
				ctx,
				&workerports.GetTrainingCourseByIDRequest{
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
		&workerports.TrainingCourseSelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		trainingCourseSelectOptionItem,
	)
}

func trainingCourseSelectOptionItem(
	entity *workertraining.TrainingCourse,
) selectOptionConnectionItem {
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

func (r *Deps) resolvePerformanceReviewTemplateSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.PerformanceReviewService.GetTemplate(
				ctx,
				&workerports.GetReviewTemplateByIDRequest{
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
		&workerports.ReviewTemplateSelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		performanceReviewTemplateSelectOptionItem,
	)
}

func performanceReviewTemplateSelectOptionItem(
	entity *workerreview.PerformanceReviewTemplate,
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

func (r *Deps) resolvePTOPolicySelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.PtoPolicyService.Get(ctx, &workerports.GetPTOPolicyByIDRequest{
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
		&workerports.PTOPolicySelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		ptoPolicySelectOptionItem,
	)
}

func ptoPolicySelectOptionItem(entity *pto.PTOPolicy) selectOptionConnectionItem {
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

func (r *Deps) resolveBenefitPlanSelectOptions(
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
		&driverpayports.BenefitPlanSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			PlanYear:           selectOptionInt16Filter(req.Filters, "planYear"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
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
func ptoPolicyTrackedTypes(entity *pto.PTOPolicy) []string {
	types := make([]string, 0, len(entity.Rules))
	for _, rule := range entity.Rules {
		types = append(types, string(rule.PTOType))
	}

	return types
}
