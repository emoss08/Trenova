package agentqualityrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality/agentqualitydb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	controlEntity            = "AgentQualityControl"
	defaultScheduleTargetCap = 200
	maxScheduleTargetCap     = 1000
)

var controlErrors = dberror.Mapping{Entity: controlEntity}

type ControlParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type controlRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewControls(p ControlParams) agentqualityports.AgentQualityControlRepository {
	return &controlRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentqualitycontrol-repository"),
	}
}

func (r *controlRepository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*agentquality.Control, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentquality.Control, error) {
		entity, err := agentqualitydb.Controls.Cache(resultcache.Settings()...).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, controlErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *controlRepository) Upsert(
	ctx context.Context,
	entity *agentquality.Control,
) (*agentquality.Control, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentquality.Control, error) {
		existing, err := r.Get(ctx, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}

		c := agentqualitydb.Controls
		if existing == nil {
			entity.ID = pulid.Nil
			entity.Version = 0
			if err = c.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, dberror.CreateVersionMismatchError(controlEntity, "")
				}
				r.l.Error("failed to create agent quality control", zap.Error(err))

				return nil, fmt.Errorf("create agent quality control: %w", err)
			}

			return entity, nil
		}

		entity.ID = existing.ID
		entity.CreatedAt = existing.CreatedAt
		err = c.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			c.Enabled,
			c.RunHourLocal,
			c.Timezone,
			c.MaxCasesPerAgent,
			c.NightlyBudgetUSD,
			c.MonthlyBudgetUSD,
			c.JudgeEnabled,
			c.JudgeSampleRate,
			c.RegressionThreshold,
			c.MinCases,
			c.ForceRerunDays,
		)
		if err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, controlErrors.Map(err, existing.ID.String())
			}

			return nil, fmt.Errorf("update agent quality control: %w", err)
		}

		return entity, nil
	})
}

type scheduleTargetRow struct {
	OrganizationID       pulid.ID
	BusinessUnitID       pulid.ID
	OrganizationName     string
	OrganizationTimezone string
	HasActiveCases       bool
}

func (row *scheduleTargetRow) target() *agentqualityports.QualityScheduleTarget {
	return &agentqualityports.QualityScheduleTarget{
		OrganizationID:       row.OrganizationID,
		BusinessUnitID:       row.BusinessUnitID,
		OrganizationName:     row.OrganizationName,
		OrganizationTimezone: row.OrganizationTimezone,
		HasActiveCases:       row.HasActiveCases,
	}
}

func scheduleTargetQuery(allTenants bool) *keel.Query[*tenant.Organization] {
	org := tenantdb.Organizations
	cases := agentqualitydb.EvalCases.Query()
	q := org.Query()
	if allTenants {
		cases = cases.AllTenants()
		q = q.AllTenants()
	}
	ec := agentqualitydb.EvalCases

	return q.Pick(
		org.ID,
		org.BusinessUnitID,
		org.Name,
		org.Timezone,
		keel.Exists(cases.Where(
			ec.OrganizationID.EqCol(org.ID),
			ec.BusinessUnitID.EqCol(org.BusinessUnitID),
			ec.Status.Eq(agentquality.CaseStatusActive),
		).Limit(1)),
	)
}

func (r *controlRepository) ScheduleTarget(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*agentqualityports.QualityScheduleTarget, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentqualityports.QualityScheduleTarget, error) {
			org := tenantdb.Organizations
			row, err := keel.IntoOne[scheduleTargetRow](ctx, r.db.Keel(ctx),
				scheduleTargetQuery(false).Where(org.ID.Eq(tenantInfo.OrgID)).Limit(1))
			if err != nil {
				return nil, dberror.HandleNotFoundError(err, "Organization")
			}

			target := row.target()
			control, err := r.Get(ctx, tenantInfo)
			switch {
			case err == nil:
				target.Control = control
			case !errortypes.IsNotFoundError(err):
				return nil, err
			}

			return target, nil
		},
	)
}

func (r *controlRepository) ListScheduleTargets(
	ctx context.Context,
	req agentqualityports.ListQualityScheduleTargetsRequest,
) ([]*agentqualityports.QualityScheduleTarget, error) {
	ctx = dbscope.WithSystem(ctx, "list agent quality schedules across every organization")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentqualityports.QualityScheduleTarget, error) {
			org := tenantdb.Organizations
			limit := clampLimit(req.Limit, defaultScheduleTargetCap, maxScheduleTargetCap)

			rows, err := keel.Into[scheduleTargetRow](
				ctx,
				r.db.Keel(ctx),
				scheduleTargetQuery(true).
					Where(keel.When(req.AfterOrganizationID.IsNotNil(), org.ID.Gt(req.AfterOrganizationID))).
					Order(org.ID.Asc()).
					Limit(limit),
			)
			if err != nil {
				return nil, fmt.Errorf("list organizations to schedule: %w", err)
			}

			targets := make([]*agentqualityports.QualityScheduleTarget, 0, len(rows))
			orgIDs := make([]pulid.ID, 0, len(rows))
			for i := range rows {
				targets = append(targets, rows[i].target())
				orgIDs = append(orgIDs, rows[i].OrganizationID)
			}
			if len(targets) == 0 {
				return targets, nil
			}

			c := agentqualitydb.Controls
			controls, err := c.AllTenants().
				Where(c.OrganizationID.In(orgIDs...)).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("read quality controls to schedule: %w", err)
			}

			byTenant := make(map[pagination.TenantInfo]*agentquality.Control, len(controls))
			for _, control := range controls {
				byTenant[pagination.TenantInfo{
					OrgID: control.OrganizationID,
					BuID:  control.BusinessUnitID,
				}] = control
			}
			for _, target := range targets {
				target.Control = byTenant[target.TenantInfo()]
			}

			return targets, nil
		},
	)
}
