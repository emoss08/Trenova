package agentdefinitionrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type VersionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type versionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewVersionRepository(p VersionParams) repositories.AgentDefinitionVersionRepository {
	return &versionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdefinition-version-repository"),
	}
}

func (r *versionRepository) Create(ctx context.Context, version *agentdefinition.DefinitionVersion) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(version).Exec(ctx); err != nil {
			return fmt.Errorf("record agent version: %w", err)
		}
		return nil
	})
}

func (r *versionRepository) List(
	ctx context.Context,
	req *repositories.ListAgentDefinitionVersionsRequest,
) ([]*agentdefinition.DefinitionVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdefinition.DefinitionVersion, error) {
		cols := buncolgen.DefinitionVersionColumns
		limit := req.Limit
		if limit <= 0 || limit > agentdefinition.VersionsKept {
			limit = agentdefinition.VersionsKept
		}

		versions := make([]*agentdefinition.DefinitionVersion, 0, limit)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&versions).
			ExcludeColumn(cols.Snapshot.Bare()).
			Relation(buncolgen.DefinitionVersionRelations.Author).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.DefinitionVersionScopeTenant(sq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
			}).
			Order(cols.Version.OrderDesc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list agent versions: %w", err)
		}

		return versions, nil
	})
}

func (r *versionRepository) Get(
	ctx context.Context,
	req *repositories.GetAgentDefinitionVersionRequest,
) (*agentdefinition.DefinitionVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentdefinition.DefinitionVersion, error) {
		cols := buncolgen.DefinitionVersionColumns
		version := new(agentdefinition.DefinitionVersion)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(version).
			Relation(buncolgen.DefinitionVersionRelations.Author).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.DefinitionVersionScopeTenant(sq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID).
					Where(cols.Version.Eq(), req.Version)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Agent version")
		}

		return version, nil
	})
}

func (r *versionRepository) LatestAt(
	ctx context.Context,
	req *repositories.GetAgentDefinitionVersionAtRequest,
) (*agentdefinition.DefinitionVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentdefinition.DefinitionVersion, error) {
		cols := buncolgen.DefinitionVersionColumns
		version := new(agentdefinition.DefinitionVersion)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(version).
			Relation(buncolgen.DefinitionVersionRelations.Author).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.DefinitionVersionScopeTenant(sq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID).
					Where(cols.Version.Lte(), req.Version)
			}).
			Order(cols.Version.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // an agent saved before versions were kept has none
		}
		if err != nil {
			return nil, fmt.Errorf("read agent version: %w", err)
		}

		return version, nil
	})
}
