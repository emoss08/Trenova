package agentextensionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const extensionEntityName = "Agent extension"

var extensionErrors = dberror.Mapping{Entity: extensionEntityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentextensionports.AgentExtensionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-extension-repository"),
	}
}

func (r *repository) ListByTenant(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*agentextension.Extension, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentextension.Extension, error) {
		e := agentextensiondb.Extensions
		entities, err := e.Query().
			Order(e.Type.Asc()).
			Limit(len(agentextension.AllTypes())).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent extensions", zap.Error(err))
			return nil, fmt.Errorf("list agent extensions: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetByType(
	ctx context.Context,
	_ pagination.TenantInfo,
	typ agentextension.Type,
) (*agentextension.Extension, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.Extension, error) {
		e := agentextensiondb.Extensions
		entity, err := e.Where(e.Type.Eq(typ)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, extensionErrors.Map(err, string(typ))
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.Extension, error) {
		e := agentextensiondb.Extensions
		inserted, err := e.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*agentextension.Extension{entity},
			e.OrganizationID,
			e.BusinessUnitID,
			e.Type,
		)
		if err != nil {
			r.l.Error("failed to create agent extension", zap.Error(err))
			return nil, fmt.Errorf("create agent extension: %w", err)
		}
		if inserted == 0 {
			return nil, errortypes.NewConflictError(
				"This extension was set up by someone else while you were editing. Reload and try again.",
			)
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentextension.Extension,
) (*agentextension.Extension, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.Extension, error) {
		if err := agentextensiondb.Extensions.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, extensionErrors.Map(err, entity.ID.String())
			}
			r.l.Error("failed to update agent extension", zap.Error(err))
			return nil, fmt.Errorf("update agent extension: %w", err)
		}

		return entity, nil
	})
}
