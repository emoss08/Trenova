package agentcontrolrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.AgentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	entity, err := r.selectControl(ctx, tenantInfo)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, dberror.HandleNotFoundError(err, "AgentControl")
	}

	control := &tenant.AgentControl{
		BusinessUnitID:         tenantInfo.BuID,
		OrganizationID:         tenantInfo.OrgID,
		ShadowMode:             true,
		BillingAgentEnabled:    false,
		DecisionTimeoutSeconds: 86400,
	}
	if _, err = r.db.DBForContext(ctx).
		NewInsert().
		Model(control).
		On("CONFLICT (organization_id, business_unit_id) DO NOTHING").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create default agent control: %w", err)
	}

	entity, err = r.selectControl(ctx, tenantInfo)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentControl")
	}

	return entity, nil
}

func (r *repository) selectControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	entity := new(tenant.AgentControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentControlScopeTenant(sq, tenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.AgentControl,
) (*tenant.AgentControl, error) {
	cols := buncolgen.AgentControlColumns

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AgentControlScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), entity.Version)
		}).
		Set(cols.ShadowMode.Set(), entity.ShadowMode).
		Set(cols.BillingAgentEnabled.Set(), entity.BillingAgentEnabled).
		Set(cols.DecisionTimeoutSeconds.Set(), entity.DecisionTimeoutSeconds).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update agent control: %w", err)
	}

	if err = dberror.CheckRowsAffected(res, "AgentControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetOrCreate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
}
