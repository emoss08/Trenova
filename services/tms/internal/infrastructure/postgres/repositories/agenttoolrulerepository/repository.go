package agenttoolrulerepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const entityName = "Tool rule"

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.AgentToolRuleOverrideRepository {
	return &repository{db: p.DB}
}

func (r *repository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agent.ToolRuleOverride, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.ToolRuleOverride, error) {
		overrides := make([]*agent.ToolRuleOverride, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&overrides).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ToolRuleOverrideScopeTenant(sq, tenantInfo)
			}).
			OrderExpr(buncolgen.ToolRuleOverrideColumns.ToolName.OrderAsc()).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list tool rule overrides: %w", err)
		}

		return overrides, nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetToolRuleOverrideRequest,
) (*agent.ToolRuleOverride, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.ToolRuleOverride, error) {
		override := new(agent.ToolRuleOverride)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(override).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ToolRuleOverrideScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.ToolRuleOverrideColumns.ToolName.Eq(), req.ToolName)
			}).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // a tool no organization rule holds has none
		}
		if err != nil {
			return nil, fmt.Errorf("get tool rule override: %w", err)
		}

		return override, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	override *agent.ToolRuleOverride,
) (*agent.ToolRuleOverride, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.ToolRuleOverride, error) {
		override.Version = 0
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			_, insertErr := r.db.DBForContext(ctx).NewInsert().Model(override).Exec(ctx)
			return insertErr
		})
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError(entityName, override.ToolName)
		}
		if err != nil {
			return nil, fmt.Errorf("create tool rule override: %w", err)
		}

		return override, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	override *agent.ToolRuleOverride,
) (*agent.ToolRuleOverride, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.ToolRuleOverride, error) {
		cols := buncolgen.ToolRuleOverrideColumns
		loaded := override.Version
		override.Version++
		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(override).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ToolRuleOverrideScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: override.OrganizationID,
					BuID:  override.BusinessUnitID,
				}).
					Where(cols.ID.Eq(), override.ID).
					Where(cols.Version.Eq(), loaded)
			}).
			Set(cols.MaxTier.Set(), override.MaxTier).
			Set(cols.ReadsExternal.Set(), override.ReadsExternal).
			Set(cols.Reason.Set(), override.Reason).
			Set(cols.UpdatedByID.Set(), override.UpdatedByID).
			Set(cols.Version.Set(), override.Version).
			Set(cols.UpdatedAt.Set(), override.UpdatedAt).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update tool rule override: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, entityName, override.ToolName); err != nil {
			return nil, err
		}

		return override, nil
	})
}
