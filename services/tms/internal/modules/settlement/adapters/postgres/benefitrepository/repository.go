package benefitrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPlanPageSize       = 200
	defaultEnrollmentPageSize = 500
)

var (
	planErrors = dberror.Mapping{
		Entity: "BenefitPlan",
		Unique: []dberror.Constraint{
			{
				Name:    "uq_benefit_plans_code_year",
				Field:   "code",
				Message: "Benefit plan with this code already exists for the plan year",
			},
		},
	}
	enrollmentErrors = dberror.Mapping{
		Entity: "WorkerBenefitEnrollment",
		Unique: []dberror.Constraint{
			{
				Name:    "uq_worker_benefit_enrollments_open",
				Field:   "benefitPlanId",
				Message: "Worker already has an open enrollment in this benefit plan",
			},
		},
	}
	openEnrollmentStatuses = []driverpay.BenefitEnrollmentStatus{
		driverpay.EnrollmentPending,
		driverpay.EnrollmentActive,
	}
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

func New(p Params) driverpayports.BenefitRepository {
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

func when(include bool, p keel.Pred) keel.Pred {
	if !include {
		return nil
	}

	return p
}

func (r *repository) ListPlans(
	ctx context.Context,
	req *driverpayports.ListBenefitPlansRequest,
) ([]*driverpay.BenefitPlan, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.BenefitPlan, error) {
		bp := driverpaydb.BenefitPlans
		q := bp.Where(
			when(req.ActiveOnly, bp.Status.Eq(domaintypes.StatusActive)),
			when(req.PlanYear > 0, bp.PlanYear.Eq(req.PlanYear)),
		).
			Order(bp.PlanYear.Desc(), bp.PlanType.Asc(), bp.Name.Asc()).
			Limit(limitOr(req.Limit, defaultPlanPageSize))
		if req.IncludePayCode {
			q = q.Load(bp.PayCode())
		}

		entities, err := q.List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list benefit plans", zap.Error(err))
			return nil, fmt.Errorf("list benefit plans: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) PlanSelectOptions(
	ctx context.Context,
	req *driverpayports.BenefitPlanSelectOptionsRequest,
) (*pagination.ListResult[*driverpay.BenefitPlan], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driverpay.BenefitPlan], error) {
			bp := driverpaydb.BenefitPlans
			q := bp.Pick(
				bp.ID,
				bp.Code,
				bp.Name,
				bp.Description,
				bp.PlanType,
				bp.Carrier,
				bp.PlanYear,
				bp.EmployeeCostMinor,
				bp.EmployerCostMinor,
				bp.CurrencyCode,
				bp.WaitingPeriodDays,
				bp.CreatedAt,
			).Where(
				bp.Status.Eq(domaintypes.StatusActive),
				when(req.PlanYear > 0, bp.PlanYear.Eq(req.PlanYear)),
			).Order(bp.PlanYear.Desc(), bp.PlanType.Asc(), bp.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						bp.Code.ContainsFold(s),
						bp.Name.ContainsFold(s),
						bp.Description.ContainsFold(s),
						bp.Carrier.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetPlanByID(
	ctx context.Context,
	req *driverpayports.GetBenefitPlanByIDRequest,
) (*driverpay.BenefitPlan, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.BenefitPlan, error) {
		bp := driverpaydb.BenefitPlans
		q := bp.Where(bp.ID.Eq(req.ID))
		if req.IncludePayCode {
			q = q.Load(bp.PayCode())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, planErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreatePlan(
	ctx context.Context,
	entity *driverpay.BenefitPlan,
) (*driverpay.BenefitPlan, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.BenefitPlan, error) {
		if err := driverpaydb.BenefitPlans.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create benefit plan", zap.Error(err))
			return nil, planErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdatePlan(
	ctx context.Context,
	entity *driverpay.BenefitPlan,
) (*driverpay.BenefitPlan, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.BenefitPlan, error) {
		if err := driverpaydb.BenefitPlans.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update benefit plan", zap.Error(err))
			return nil, planErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CountPlanEnrollments(
	ctx context.Context,
	_ pagination.TenantInfo,
	planID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		e := driverpaydb.WorkerBenefitEnrollments
		total, err := e.Where(
			e.BenefitPlanID.Eq(planID),
			e.Status.In(openEnrollmentStatuses...),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count plan enrollments", zap.Error(err))
			return 0, fmt.Errorf("count plan enrollments: %w", err)
		}

		return int(total), nil
	})
}

func (r *repository) ListEnrollments(
	ctx context.Context,
	req *driverpayports.ListBenefitEnrollmentsRequest,
) ([]*driverpay.WorkerBenefitEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driverpay.WorkerBenefitEnrollment, error) {
			e := driverpaydb.WorkerBenefitEnrollments
			q := e.Where(
				when(!req.WorkerID.IsNil(), e.WorkerID.Eq(req.WorkerID)),
				when(!req.PlanID.IsNil(), e.BenefitPlanID.Eq(req.PlanID)),
				when(req.OpenOnly, e.Status.In(openEnrollmentStatuses...)),
				when(len(req.Statuses) > 0, e.Status.In(req.Statuses...)),
			).
				Order(e.EffectiveFrom.Desc()).
				Limit(limitOr(req.Limit, defaultEnrollmentPageSize))
			if req.IncludePlan {
				q = q.Load(e.BenefitPlan())
			}
			if req.IncludeWorker {
				q = q.Load(e.Worker())
			}

			entities, err := q.List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list benefit enrollments", zap.Error(err))
				return nil, fmt.Errorf("list benefit enrollments: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) GetEnrollmentByID(
	ctx context.Context,
	req *driverpayports.GetBenefitEnrollmentByIDRequest,
) (*driverpay.WorkerBenefitEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*driverpay.WorkerBenefitEnrollment, error) {
			e := driverpaydb.WorkerBenefitEnrollments
			q := e.Where(e.ID.Eq(req.ID))
			if req.IncludePlan {
				q = q.Load(e.BenefitPlan())
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, enrollmentErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateEnrollment(
	ctx context.Context,
	entity *driverpay.WorkerBenefitEnrollment,
) (*driverpay.WorkerBenefitEnrollment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*driverpay.WorkerBenefitEnrollment, error) {
			err := driverpaydb.WorkerBenefitEnrollments.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create benefit enrollment", zap.Error(err))
				return nil, enrollmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateEnrollment(
	ctx context.Context,
	entity *driverpay.WorkerBenefitEnrollment,
) (*driverpay.WorkerBenefitEnrollment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*driverpay.WorkerBenefitEnrollment, error) {
			err := driverpaydb.WorkerBenefitEnrollments.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update benefit enrollment", zap.Error(err))
				return nil, enrollmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) BenefitCosts(
	ctx context.Context,
	_ pagination.TenantInfo,
	planYear int16,
) ([]driverpayports.BenefitCostRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]driverpayports.BenefitCostRow, error) {
		e := driverpaydb.WorkerBenefitEnrollments
		bp := driverpaydb.BenefitPlans
		active := e.Status.Eq(driverpay.EnrollmentActive)
		enrolled := keel.Count().Filter(active)

		rows, err := keel.Into[driverpayports.BenefitCostRow](
			ctx,
			r.db.Keel(ctx),
			e.Join(bp, bp.ID.EqCol(e.BenefitPlanID)).
				Where(when(planYear > 0, bp.PlanYear.Eq(planYear))).
				Group(bp.ID, bp.Name, bp.PlanType).
				Pick(
					bp.ID,
					bp.Name,
					keel.DynOf(bp.PlanType).Cast(keel.TypeString),
					enrolled,
					keel.Count().Filter(e.Status.Eq(driverpay.EnrollmentWaived)),
					keel.DynOf(e.EmployeeCostMinor.Sum().Filter(active)).Coalesce(0),
					keel.DynOf(e.EmployerCostMinor.Sum().Filter(active)).Coalesce(0),
				).
				Order(enrolled.Desc(), bp.Name.Asc()),
		)
		if err != nil {
			r.l.Error("failed to total benefit costs", zap.Error(err))
			return nil, fmt.Errorf("total benefit costs: %w", err)
		}

		return rows, nil
	})
}
