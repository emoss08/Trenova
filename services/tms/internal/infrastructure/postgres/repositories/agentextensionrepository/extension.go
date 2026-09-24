package agentextensionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	extensionEntityName = "Agent extension"
	extensionConflict   = "CONFLICT (organization_id, business_unit_id, type) DO NOTHING"
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

func New(p Params) repositories.AgentExtensionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-extension-repository"),
	}
}

func (r *repository) ListByTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agentextension.Extension, error) {
	entities := make([]*agentextension.Extension, 0, len(agentextension.AllTypes()))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.ExtensionApplyTenant(tenantInfo)).
		Order(buncolgen.ExtensionColumns.Type.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list agent extensions", zap.Error(err))
		return nil, fmt.Errorf("list agent extensions: %w", err)
	}

	return entities, nil
}

func (r *repository) GetByType(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*agentextension.Extension, error) {
	entity := new(agentextension.Extension)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtensionScopeTenant(sq, tenantInfo).
				Where(buncolgen.ExtensionColumns.Type.Eq(), typ)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, extensionEntityName)
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	result, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(extensionConflict).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to create agent extension", zap.Error(err))
		return nil, fmt.Errorf("create agent extension: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("create agent extension rows affected: %w", err)
	}
	if affected == 0 {
		return nil, errortypes.NewConflictError(
			"This extension was set up by someone else while you were editing. Reload and try again.",
		)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	ov := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ExtensionColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update agent extension", zap.Error(err))
		return nil, fmt.Errorf("update agent extension: %w", err)
	}
	if err = dberror.CheckRowsAffected(result, extensionEntityName, entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
