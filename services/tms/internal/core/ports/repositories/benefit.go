package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListBenefitPlansRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActiveOnly bool                  `json:"activeOnly"`
	// PlanYear narrows to one year; zero returns every year on file.
	PlanYear       int16 `json:"planYear"`
	IncludePayCode bool  `json:"includePayCode"`
	Limit          int   `json:"limit"`
}

type GetBenefitPlanByIDRequest struct {
	ID             pulid.ID              `json:"id"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	IncludePayCode bool                  `json:"includePayCode"`
}

type ListBenefitEnrollmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	PlanID     pulid.ID              `json:"planId"`
	// OpenOnly drops waived and ended enrollments, which is what a payroll run
	// wants; a benefits administrator reading somebody's history wants them all.
	OpenOnly bool `json:"openOnly"`
	// Statuses narrows to the states named; empty means every state.
	Statuses      []driverpay.BenefitEnrollmentStatus `json:"statuses"`
	IncludePlan   bool                                `json:"includePlan"`
	IncludeWorker bool                                `json:"includeWorker"`
	Limit         int                                 `json:"limit"`
}

type GetBenefitEnrollmentByIDRequest struct {
	ID          pulid.ID              `json:"id"`
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	IncludePlan bool                  `json:"includePlan"`
}

// BenefitCostRow is one plan's cost across the organisation, for the enrolment
// summary the benefits administrator reads.
type BenefitCostRow struct {
	PlanID            pulid.ID `bun:"plan_id"`
	PlanName          string   `bun:"plan_name"`
	PlanType          string   `bun:"plan_type"`
	Enrolled          int      `bun:"enrolled"`
	Waived            int      `bun:"waived"`
	EmployeeCostMinor int64    `bun:"employee_cost_minor"`
	EmployerCostMinor int64    `bun:"employer_cost_minor"`
}

type BenefitRepository interface {
	ListPlans(ctx context.Context, req *ListBenefitPlansRequest) ([]*driverpay.BenefitPlan, error)
	GetPlanByID(
		ctx context.Context,
		req *GetBenefitPlanByIDRequest,
	) (*driverpay.BenefitPlan, error)
	CreatePlan(
		ctx context.Context,
		entity *driverpay.BenefitPlan,
	) (*driverpay.BenefitPlan, error)
	UpdatePlan(
		ctx context.Context,
		entity *driverpay.BenefitPlan,
	) (*driverpay.BenefitPlan, error)
	// CountPlanEnrollments answers how many people are on a plan, which is what
	// stops one being archived out from under them.
	CountPlanEnrollments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		planID pulid.ID,
	) (int, error)

	ListEnrollments(
		ctx context.Context,
		req *ListBenefitEnrollmentsRequest,
	) ([]*driverpay.WorkerBenefitEnrollment, error)
	GetEnrollmentByID(
		ctx context.Context,
		req *GetBenefitEnrollmentByIDRequest,
	) (*driverpay.WorkerBenefitEnrollment, error)
	CreateEnrollment(
		ctx context.Context,
		entity *driverpay.WorkerBenefitEnrollment,
	) (*driverpay.WorkerBenefitEnrollment, error)
	UpdateEnrollment(
		ctx context.Context,
		entity *driverpay.WorkerBenefitEnrollment,
	) (*driverpay.WorkerBenefitEnrollment, error)

	BenefitCosts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		planYear int16,
	) ([]BenefitCostRow, error)
}
