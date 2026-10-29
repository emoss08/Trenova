package agenttoolrulerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust/agenttooltrustdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
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

func New(p Params) agenttoolports.AgentToolRuleOverrideRepository {
	return &repository{db: p.DB}
}

func (r *repository) List(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*agenttooltrust.ToolRuleOverride, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agenttooltrust.ToolRuleOverride, error) {
			o := agenttooltrustdb.ToolRuleOverrides
			overrides, err := o.Query().
				Order(o.ToolName.Asc()).
				Cache(resultcache.Settings()...).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list tool rule overrides: %w", err)
			}

			return overrides, nil
		},
	)
}

func (r *repository) Get(
	ctx context.Context,
	req agenttoolports.GetToolRuleOverrideRequest,
) (*agenttooltrust.ToolRuleOverride, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agenttooltrust.ToolRuleOverride, error) {
			o := agenttooltrustdb.ToolRuleOverrides
			override, err := o.Where(o.ToolName.Eq(req.ToolName)).One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // a tool no organization rule holds has none
			}
			if err != nil {
				return nil, fmt.Errorf("get tool rule override: %w", err)
			}

			return override, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	override *agenttooltrust.ToolRuleOverride,
) (*agenttooltrust.ToolRuleOverride, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*agenttooltrust.ToolRuleOverride, error) {
			override.Version = 0
			err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
				return agenttooltrustdb.ToolRuleOverrides.Create(ctx, r.db.Keel(ctx), override)
			})
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, dberror.CreateVersionMismatchError(entityName, override.ToolName)
			}
			if err != nil {
				return nil, fmt.Errorf("create tool rule override: %w", err)
			}

			return override, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	override *agenttooltrust.ToolRuleOverride,
) (*agenttooltrust.ToolRuleOverride, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*agenttooltrust.ToolRuleOverride, error) {
			o := agenttooltrustdb.ToolRuleOverrides
			err := o.Save(
				ctx,
				r.db.Keel(ctx),
				override,
				o.MaxTier,
				o.ReadsExternal,
				o.Reason,
				o.UpdatedByID,
			)
			if dberror.IsVersionConflict(err) {
				return nil, dberror.CreateVersionMismatchError(entityName, override.ToolName)
			}
			if err != nil {
				return nil, fmt.Errorf("update tool rule override: %w", err)
			}

			return override, nil
		},
	)
}
