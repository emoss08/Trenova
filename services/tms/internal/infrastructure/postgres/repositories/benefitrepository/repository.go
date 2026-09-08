package benefitrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPlanPageSize       = 200
	defaultEnrollmentPageSize = 500
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.BenefitRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.benefit-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListPlans(
	ctx context.Context,
	req *repositories.ListBenefitPlansRequest,
) ([]*driverpay.BenefitPlan, error) {
	cols := buncolgen.BenefitPlanColumns
	entities := make([]*driverpay.BenefitPlan, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.BenefitPlanScopeTenant(sq, req.TenantInfo)
			if req.ActiveOnly {
				sq = sq.Where(cols.Status.Eq(), domaintypes.StatusActive)
			}
			if req.PlanYear > 0 {
				sq = sq.Where(cols.PlanYear.Eq(), req.PlanYear)
			}
			return sq
		}).
		Order(cols.PlanYear.OrderDesc()).
		Order(cols.PlanType.OrderAsc()).
		Order(cols.Name.OrderAsc()).
		Limit(limitOr(req.Limit, defaultPlanPageSize))

	if req.IncludePayCode {
		q = q.Relation(buncolgen.BenefitPlanRelations.PayCode)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list benefit plans", zap.Error(err))
		return nil, fmt.Errorf("list benefit plans: %w", err)
	}

	return entities, nil
}

func (r *repository) GetPlanByID(
	ctx context.Context,
	req *repositories.GetBenefitPlanByIDRequest,
) (*driverpay.BenefitPlan, error) {
	entity := new(driverpay.BenefitPlan)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.BenefitPlanScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.BenefitPlanColumns.ID.Eq(), req.ID)
		})
	if req.IncludePayCode {
		q = q.Relation(buncolgen.BenefitPlanRelations.PayCode)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "BenefitPlan")
	}

	return entity, nil
}

func (r *repository) CreatePlan(
	ctx context.Context,
	entity *driverpay.BenefitPlan,
) (*driverpay.BenefitPlan, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create benefit plan", zap.Error(err))
		return nil, fmt.Errorf("create benefit plan: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdatePlan(
	ctx context.Context,
	entity *driverpay.BenefitPlan,
) (*driverpay.BenefitPlan, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.BenefitPlanColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update benefit plan", zap.Error(err))
		return nil, fmt.Errorf("update benefit plan: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "BenefitPlan", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CountPlanEnrollments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	planID pulid.ID,
) (int, error) {
	cols := buncolgen.WorkerBenefitEnrollmentColumns
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driverpay.WorkerBenefitEnrollment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerBenefitEnrollmentScopeTenant(sq, tenantInfo).
				Where(cols.BenefitPlanID.Eq(), planID).
				Where(cols.Status.In(), bun.In([]driverpay.BenefitEnrollmentStatus{
					driverpay.EnrollmentPending,
					driverpay.EnrollmentActive,
				}))
		}).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count plan enrollments", zap.Error(err))
		return 0, fmt.Errorf("count plan enrollments: %w", err)
	}

	return total, nil
}

func (r *repository) ListEnrollments(
	ctx context.Context,
	req *repositories.ListBenefitEnrollmentsRequest,
) ([]*driverpay.WorkerBenefitEnrollment, error) {
	cols := buncolgen.WorkerBenefitEnrollmentColumns
	entities := make([]*driverpay.WorkerBenefitEnrollment, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerBenefitEnrollmentScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if !req.PlanID.IsNil() {
				sq = sq.Where(cols.BenefitPlanID.Eq(), req.PlanID)
			}
			if req.OpenOnly {
				sq = sq.Where(cols.Status.In(), bun.In([]driverpay.BenefitEnrollmentStatus{
					driverpay.EnrollmentPending,
					driverpay.EnrollmentActive,
				}))
			}
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.In(req.Statuses))
			}
			return sq
		}).
		Order(cols.EffectiveFrom.OrderDesc()).
		Limit(limitOr(req.Limit, defaultEnrollmentPageSize))

	if req.IncludePlan {
		q = q.Relation(buncolgen.WorkerBenefitEnrollmentRelations.BenefitPlan)
	}
	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerBenefitEnrollmentRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list benefit enrollments", zap.Error(err))
		return nil, fmt.Errorf("list benefit enrollments: %w", err)
	}

	return entities, nil
}

func (r *repository) GetEnrollmentByID(
	ctx context.Context,
	req *repositories.GetBenefitEnrollmentByIDRequest,
) (*driverpay.WorkerBenefitEnrollment, error) {
	entity := new(driverpay.WorkerBenefitEnrollment)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerBenefitEnrollmentScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerBenefitEnrollmentColumns.ID.Eq(), req.ID)
		})
	if req.IncludePlan {
		q = q.Relation(buncolgen.WorkerBenefitEnrollmentRelations.BenefitPlan)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerBenefitEnrollment")
	}

	return entity, nil
}

func (r *repository) CreateEnrollment(
	ctx context.Context,
	entity *driverpay.WorkerBenefitEnrollment,
) (*driverpay.WorkerBenefitEnrollment, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create benefit enrollment", zap.Error(err))
		return nil, fmt.Errorf("create benefit enrollment: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateEnrollment(
	ctx context.Context,
	entity *driverpay.WorkerBenefitEnrollment,
) (*driverpay.WorkerBenefitEnrollment, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerBenefitEnrollmentColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update benefit enrollment", zap.Error(err))
		return nil, fmt.Errorf("update benefit enrollment: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"WorkerBenefitEnrollment",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

// BenefitCosts is what each plan is costing across the organisation. Grouped
// SQL rather than a walk: the answer is a handful of rows however many people
// are enrolled.
func (r *repository) BenefitCosts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	planYear int16,
) ([]repositories.BenefitCostRow, error) {
	rows := make([]repositories.BenefitCostRow, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driverpay.WorkerBenefitEnrollment)(nil)).
		Join("JOIN benefit_plans AS bplan").
		JoinOn("bplan.id = wben.benefit_plan_id").
		JoinOn("bplan.organization_id = wben.organization_id").
		JoinOn("bplan.business_unit_id = wben.business_unit_id").
		Where("wben.organization_id = ?", tenantInfo.OrgID).
		Where("wben.business_unit_id = ?", tenantInfo.BuID).
		ColumnExpr("bplan.id AS plan_id").
		ColumnExpr("bplan.name AS plan_name").
		ColumnExpr("bplan.plan_type::text AS plan_type").
		ColumnExpr("COUNT(*) FILTER (WHERE wben.status = 'Active') AS enrolled").
		ColumnExpr("COUNT(*) FILTER (WHERE wben.status = 'Waived') AS waived").
		ColumnExpr(
			"COALESCE(SUM(wben.employee_cost_minor) FILTER (WHERE wben.status = 'Active'), 0) AS employee_cost_minor",
		).
		ColumnExpr(
			"COALESCE(SUM(wben.employer_cost_minor) FILTER (WHERE wben.status = 'Active'), 0) AS employer_cost_minor",
		).
		GroupExpr("bplan.id, bplan.name, bplan.plan_type").
		OrderExpr("enrolled DESC, plan_name")

	if planYear > 0 {
		q = q.Where("bplan.plan_year = ?", planYear)
	}

	if err := q.Scan(ctx, &rows); err != nil {
		r.l.Error("failed to total benefit costs", zap.Error(err))
		return nil, fmt.Errorf("total benefit costs: %w", err)
	}

	return rows, nil
}
