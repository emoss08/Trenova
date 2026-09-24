package agentqualityrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	controlEntity            = "AgentQualityControl"
	defaultScheduleTargetCap = 200
	maxScheduleTargetCap     = 1000
)

type ControlParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type controlRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewControls(p ControlParams) repositories.AgentQualityControlRepository {
	return &controlRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentqualitycontrol-repository"),
	}
}

func (r *controlRepository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*agentquality.Control, error) {
	entity := new(agentquality.Control)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ControlScopeTenant(sq, tenantInfo)
		}).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, controlEntity)
	}

	return entity, nil
}

func (r *controlRepository) Upsert(
	ctx context.Context,
	entity *agentquality.Control,
) (*agentquality.Control, error) {
	cols := buncolgen.ControlColumns
	dba := r.db.DBForContext(ctx)

	existing, err := r.Get(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil && !dberror.IsNotFoundError(err) {
		return nil, err
	}

	if existing == nil {
		entity.ID = pulid.Nil
		entity.Version = 0
		if _, err = dba.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, dberror.CreateVersionMismatchError(controlEntity, "")
			}
			r.l.Error("failed to create agent quality control", zap.Error(err))

			return nil, fmt.Errorf("create agent quality control: %w", err)
		}

		return entity, nil
	}

	ov := entity.Version
	entity.ID = existing.ID
	entity.CreatedAt = existing.CreatedAt
	entity.Version = ov + 1
	entity.UpdatedAt = timeutils.NowUnix()

	res, err := dba.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ControlScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), existing.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Enabled.Set(), entity.Enabled).
		Set(cols.RunHourLocal.Set(), entity.RunHourLocal).
		Set(cols.Timezone.Set(), bun.NullZero(entity.Timezone)).
		Set(cols.MaxCasesPerAgent.Set(), entity.MaxCasesPerAgent).
		Set(cols.NightlyBudgetUSD.Set(), entity.NightlyBudgetUSD).
		Set(cols.MonthlyBudgetUSD.Set(), entity.MonthlyBudgetUSD).
		Set(cols.JudgeEnabled.Set(), entity.JudgeEnabled).
		Set(cols.JudgeSampleRate.Set(), entity.JudgeSampleRate).
		Set(cols.RegressionThreshold.Set(), entity.RegressionThreshold).
		Set(cols.MinCases.Set(), entity.MinCases).
		Set(cols.ForceRerunDays.Set(), entity.ForceRerunDays).
		Set(cols.UpdatedAt.Set(), entity.UpdatedAt).
		Set(cols.Version.Set(), entity.Version).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update agent quality control: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update agent quality control rows: %w", err)
	}
	if rows == 0 {
		return nil, dberror.CreateVersionMismatchError(controlEntity, existing.ID.String())
	}

	return entity, nil
}

func (r *controlRepository) scheduleTargetQuery(
	dba bun.IDB,
) *bun.SelectQuery {
	org := buncolgen.OrganizationColumns
	cases := buncolgen.EvalCaseColumns

	activeCases := dba.NewSelect().
		Model((*agentquality.EvalCase)(nil)).
		ColumnExpr("1").
		Where(cases.OrganizationID.EqColumn(org.ID)).
		Where(cases.BusinessUnitID.EqColumn(org.BusinessUnitID)).
		Where(cases.Status.Eq(), agentquality.CaseStatusActive).
		Limit(1)

	return dba.NewSelect().
		Model((*tenant.Organization)(nil)).
		ColumnExpr(org.ID.As("organization_id")).
		ColumnExpr(org.BusinessUnitID.As("business_unit_id")).
		ColumnExpr(org.Name.As("organization_name")).
		ColumnExpr(org.Timezone.As("organization_timezone")).
		ColumnExpr("EXISTS (?) AS has_active_cases", activeCases)
}

func (r *controlRepository) ScheduleTarget(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.QualityScheduleTarget, error) {
	org := buncolgen.OrganizationColumns
	dba := r.db.DBForContext(ctx)

	target := new(repositories.QualityScheduleTarget)
	if err := r.scheduleTargetQuery(dba).
		Where(org.ID.Eq(), tenantInfo.OrgID).
		Where(org.BusinessUnitID.Eq(), tenantInfo.BuID).
		Limit(1).
		Scan(ctx, target); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Organization")
	}

	control, err := r.Get(ctx, tenantInfo)
	switch {
	case err == nil:
		target.Control = control
	case !dberror.IsNotFoundError(err):
		return nil, err
	}

	return target, nil
}

func (r *controlRepository) ListScheduleTargets(
	ctx context.Context,
	req repositories.ListQualityScheduleTargetsRequest,
) ([]*repositories.QualityScheduleTarget, error) {
	org := buncolgen.OrganizationColumns
	limit := clampLimit(req.Limit, defaultScheduleTargetCap, maxScheduleTargetCap)
	dba := r.db.DBForContext(ctx)

	q := r.scheduleTargetQuery(dba)
	if req.AfterOrganizationID.IsNotNil() {
		q = q.Where(org.ID.Gt(), req.AfterOrganizationID)
	}

	targets := make([]*repositories.QualityScheduleTarget, 0, limit)
	if err := q.OrderExpr(org.ID.OrderAsc()).Limit(limit).Scan(ctx, &targets); err != nil {
		return nil, fmt.Errorf("list organizations to schedule: %w", err)
	}
	if len(targets) == 0 {
		return targets, nil
	}

	orgIDs := make([]pulid.ID, 0, len(targets))
	for _, target := range targets {
		orgIDs = append(orgIDs, target.OrganizationID)
	}

	controls := make([]*agentquality.Control, 0, len(targets))
	if err := dba.NewSelect().
		Model(&controls).
		Where(buncolgen.ControlColumns.OrganizationID.In(), bun.List(orgIDs)).
		Scan(ctx); err != nil && !errors.Is(err, sql.ErrNoRows) {
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
}
