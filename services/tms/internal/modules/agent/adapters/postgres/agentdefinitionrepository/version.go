package agentdefinitionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var versionErrors = dberror.Mapping{Entity: "Agent version"}

type VersionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type versionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewVersionRepository(p VersionParams) agentdefinitionports.AgentDefinitionVersionRepository {
	return &versionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdefinition-version-repository"),
	}
}

func (r *versionRepository) Create(
	ctx context.Context,
	version *agentdefinition.DefinitionVersion,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := agentdefinitiondb.DefinitionVersions.Create(ctx, r.db.Keel(ctx), version)
		if err != nil {
			return fmt.Errorf("record agent version: %w", err)
		}
		return nil
	})
}

func (r *versionRepository) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionVersionsRequest,
) ([]*agentdefinition.DefinitionVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentdefinition.DefinitionVersion, error) {
			limit := req.Limit
			if limit <= 0 || limit > agentdefinition.VersionsKept {
				limit = agentdefinition.VersionsKept
			}

			v := agentdefinitiondb.DefinitionVersions
			versions, err := v.Omit(v.Snapshot).
				Load(v.Author()).
				Where(v.AgentDefinitionID.Eq(req.AgentDefinitionID)).
				Order(v.Version.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list agent versions: %w", err)
			}

			return versions, nil
		},
	)
}

func (r *versionRepository) Get(
	ctx context.Context,
	req *agentdefinitionports.GetAgentDefinitionVersionRequest,
) (*agentdefinition.DefinitionVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentdefinition.DefinitionVersion, error) {
			v := agentdefinitiondb.DefinitionVersions
			version, err := v.Load(v.Author()).
				Where(v.AgentDefinitionID.Eq(req.AgentDefinitionID), v.Version.Eq(req.Version)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, versionErrors.Map(err, req.AgentDefinitionID.String())
			}

			return version, nil
		},
	)
}

func (r *versionRepository) LatestAt(
	ctx context.Context,
	req *agentdefinitionports.GetAgentDefinitionVersionAtRequest,
) (*agentdefinition.DefinitionVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentdefinition.DefinitionVersion, error) {
			v := agentdefinitiondb.DefinitionVersions
			version, err := v.Load(v.Author()).
				Where(v.AgentDefinitionID.Eq(req.AgentDefinitionID), v.Version.Lte(req.Version)).
				Order(v.Version.Desc()).
				First(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // an agent saved before versions were kept has none
			}
			if err != nil {
				return nil, fmt.Errorf("read agent version: %w", err)
			}

			return version, nil
		},
	)
}
