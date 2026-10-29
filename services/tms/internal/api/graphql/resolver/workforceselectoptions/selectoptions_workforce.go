package workforceselectoptions

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/benefitsservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgstructureports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptopolicyports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/selfserviceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workercredentialports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerreviewports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workertrainingports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	WorkerService            *workerservice.Service
	SchedulingService        *schedulingservice.Service
	SelfServiceService       *selfserviceservice.Service
	OrgStructureService      *orgstructureservice.Service
	WorkerCredentialService  *workercredentialservice.Service
	WorkerTrainingService    *workertrainingservice.Service
	PerformanceReviewService *performancereviewservice.Service
	PtoPolicyService         *ptopolicyservice.Service
	BenefitsService          *benefitsservice.Service
}

func (r *Sources) WorkerCredentialType(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.WorkerCredentialService.GetType(
				ctx,
				&workercredentialports.GetCredentialTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, workerCredentialTypeSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.WorkerCredentialService.TypeSelectOptions(
		ctx,
		&workercredentialports.WorkerCredentialTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		workerCredentialTypeSelectOptionItem,
	)
}

func workerCredentialTypeSelectOptionItem(
	entity *workercredential.WorkerCredentialType,
) selectoption.Item {
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
			Meta: map[string]any{
				selectoption.MetaKeyCode: entity.Code,
				"category":               string(entity.Category),
				"isRequired":             entity.IsRequired,
				"requiresNumber":         entity.RequiresNumber,
				"requiresDocument":       entity.RequiresDocument,
				"profileField":           string(entity.ProfileField),
				"validityMonths":         int32Value(entity.ValidityMonths),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Sources) TrainingCourse(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.WorkerTrainingService.GetCourse(
				ctx,
				&workertrainingports.GetTrainingCourseByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, trainingCourseSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.WorkerTrainingService.CourseSelectOptions(
		ctx,
		&workertrainingports.TrainingCourseSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		trainingCourseSelectOptionItem,
	)
}

func trainingCourseSelectOptionItem(
	entity *workertraining.TrainingCourse,
) selectoption.Item {
	meta := map[string]any{
		selectoption.MetaKeyCode: entity.Code,
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

	return selectoption.ItemFor(
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

func (r *Sources) PerformanceReviewTemplate(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.PerformanceReviewService.GetTemplate(
				ctx,
				&workerreviewports.GetReviewTemplateByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, performanceReviewTemplateSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.PerformanceReviewService.TemplateSelectOptions(
		ctx,
		&workerreviewports.ReviewTemplateSelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		performanceReviewTemplateSelectOptionItem,
	)
}

func performanceReviewTemplateSelectOptionItem(
	entity *workerreview.PerformanceReviewTemplate,
) selectoption.Item {
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
			Meta: map[string]any{
				selectoption.MetaKeyCode: entity.Code,
				"isDefault":              entity.IsDefault,
				"itemCount":              len(entity.Items),
				"cadenceMonths":          int32Value(entity.CadenceMonths),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Sources) PTOPolicy(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.PtoPolicyService.Get(ctx, &ptopolicyports.GetPTOPolicyByIDRequest{
				ID:           id,
				TenantInfo:   req.TenantInfo,
				IncludeRules: true,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, ptoPolicySelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.PtoPolicyService.SelectOptions(
		ctx,
		&ptopolicyports.PTOPolicySelectOptionsRequest{SelectQueryRequest: req.SelectQuery},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		ptoPolicySelectOptionItem,
	)
}

func ptoPolicySelectOptionItem(entity *pto.PTOPolicy) selectoption.Item {
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
			Meta: map[string]any{
				selectoption.MetaKeyCode: entity.Code,
				"isDefault":              entity.IsDefault,
				"yearBasis":              string(entity.YearBasis),
				"requiresApproval":       entity.RequiresApproval,
				"ptoTypes":               ptoPolicyTrackedTypes(entity),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Sources) BenefitPlan(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.BenefitsService.GetPlan(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, benefitPlanSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.BenefitsService.PlanSelectOptions(
		ctx,
		&driverpayports.BenefitPlanSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			PlanYear:           selectoption.Int16Filter(req.Filters, "planYear"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		benefitPlanSelectOptionItem,
	)
}

func benefitPlanSelectOptionItem(entity *paybenefit.BenefitPlan) selectoption.Item {
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringutils.Ptr(entity.Description),
			Meta: map[string]any{
				selectoption.MetaKeyCode: entity.Code,
				"planType":               string(entity.PlanType),
				"planYear":               entity.PlanYear,
				"carrier":                entity.Carrier,
				"employeeCostMinor":      entity.EmployeeCostMinor,
				"employerCostMinor":      entity.EmployerCostMinor,
				"currencyCode":           entity.CurrencyCode,
				"waitingPeriodDays":      entity.WaitingPeriodDays,
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

func (r *Sources) Worker(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.WorkerService.Get(
				ctx,
				workerports.GetWorkerByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, workerSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.WorkerService.SelectOptions(
		ctx,
		&workerports.WorkerSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			OwnerOperatorsOnly: selectoption.BoolFilter(req.Filters, "ownerOperatorsOnly"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		workerSelectOptionItem,
	)
}

func workerSelectOptionItem(entity *worker.Worker) selectoption.Item {
	return selectoption.ItemFor(
		workerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func workerSelectOption(entity *worker.Worker) *gqlmodel.SelectOption {
	label := entity.WholeName
	if label == "" {
		label = entity.FullName()
	}

	meta := map[string]any{
		"firstName": entity.FirstName,
		"lastName":  entity.LastName,
		"wholeName": label,
	}
	if entity.FleetCode != nil {
		meta["fleetCode"] = entity.FleetCode.Code
	}

	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: label,
		Meta:  meta,
	}
}

func (r *Sources) ShiftTemplate(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.SchedulingService.GetTemplate(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, shiftTemplateSelectOptionItem(entity))
		}
		return selectoption.Connection(items, len(items), 0)
	}

	templates, err := r.SchedulingService.ListTemplates(
		ctx,
		&workerscheduleports.ListShiftTemplatesRequest{
			TenantInfo: req.TenantInfo,
			ActiveOnly: true,
		},
	)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))
	items := make([]selectoption.Item, 0, len(templates))
	for _, template := range templates {
		if query != "" &&
			!strings.Contains(strings.ToLower(template.Code), query) &&
			!strings.Contains(strings.ToLower(template.Name), query) {
			continue
		}
		items = append(items, shiftTemplateSelectOptionItem(template))
	}

	return selectoption.Connection(items, len(items), 0)
}

func (r *Sources) WorkerPolicy(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.SelfServiceService.GetPolicy(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, workerPolicySelectOptionItem(entity))
		}
		return selectoption.Connection(items, len(items), 0)
	}

	policies, err := r.SelfServiceService.ListPolicies(
		ctx,
		&selfserviceports.ListWorkerPoliciesRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))
	items := make([]selectoption.Item, 0, len(policies))
	for _, policy := range policies {
		if policy.Status != domaintypes.StatusActive {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(policy.Code), query) &&
			!strings.Contains(strings.ToLower(policy.Title), query) {
			continue
		}
		items = append(items, workerPolicySelectOptionItem(policy))
	}

	return selectoption.Connection(items, len(items), 0)
}

// resolveJobPositionSelectOptions offers the half of the chart the picker is
// filling: driving titles for a worker's record, front-office titles for a
// user's membership. Offering both would put somebody on the wrong roster.
func (r *Sources) JobPosition(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.OrgStructureService.GetPosition(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, jobPositionSelectOptionItem(entity))
		}
		return selectoption.Connection(items, len(items), 0)
	}

	positions, err := r.OrgStructureService.ListPositions(
		ctx,
		&orgstructureports.ListJobPositionsRequest{
			TenantInfo:  req.TenantInfo,
			ActiveOnly:  true,
			DrivingOnly: selectoption.BoolFilter(req.Filters, "drivingOnly"),
		},
	)
	if err != nil {
		return nil, err
	}

	nonDrivingOnly := selectoption.BoolFilter(req.Filters, "nonDrivingOnly")
	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))
	items := make([]selectoption.Item, 0, len(positions))
	for _, position := range positions {
		if nonDrivingOnly && position.IsDrivingPosition {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(position.Code), query) &&
			!strings.Contains(strings.ToLower(position.Title), query) {
			continue
		}
		items = append(items, jobPositionSelectOptionItem(position))
	}

	return selectoption.Connection(items, len(items), 0)
}
