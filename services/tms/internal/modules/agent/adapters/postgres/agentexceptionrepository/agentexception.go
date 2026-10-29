package agentexceptionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception/agentexceptiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentexceptionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var exceptionErrors = dberror.Mapping{Entity: "AgentException"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentexceptionports.AgentExceptionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentexception-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *agentexceptionports.ListAgentExceptionRequest,
) (*pagination.ListResult[*agentexception.AgentException], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*agentexception.AgentException], error) {
			e := agentexceptiondb.AgentExceptions
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				e.Query(),
				&listquery.Request[agentexception.AgentException]{
					Filter: req.Filter,
					Then:   []keel.Order{e.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to scan and count agent exceptions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req agentexceptionports.ListAgentExceptionsByIDsRequest,
) ([]*agentexception.AgentException, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentexception.AgentException, error) {
			if len(req.IDs) == 0 {
				return []*agentexception.AgentException{}, nil
			}

			e := agentexceptiondb.AgentExceptions
			entities, err := e.Where(e.ID.In(req.IDs...)).
				Limit(len(req.IDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list agent exceptions by ids", zap.Error(err))

				return nil, fmt.Errorf("list agent exceptions by ids: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentexceptionports.ListAgentExceptionConnectionRequest,
) (*pagination.CursorListResult[*agentexception.AgentException], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agentexception.AgentException], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentexceptiondb.AgentExceptions.Query(),
				&listquery.Request[agentexception.AgentException]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to scan agent exceptions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentexceptionports.GetAgentExceptionByIDRequest,
) (*agentexception.AgentException, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentexception.AgentException, error) {
		entity, err := agentexceptiondb.AgentExceptions.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			r.l.Error(
				"failed to get agent exception",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return nil, exceptionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentexception.AgentException,
) (*agentexception.AgentException, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentexception.AgentException, error) {
		if err := agentexceptiondb.AgentExceptions.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent exception", zap.Error(err))
			return nil, exceptionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateResolution(
	ctx context.Context,
	req *agentexceptionports.UpdateAgentExceptionResolutionRequest,
) (*agentexception.AgentException, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentexception.AgentException, error) {
		e := agentexceptiondb.AgentExceptions
		rows, err := e.Where(e.ID.Eq(req.ID)).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			e.ResolutionState.To(req.ResolutionState),
			e.ResolutionNotes.To(req.ResolutionNotes),
			e.Version.ToExpr(e.Version),
		)
		if err != nil {
			r.l.Error(
				"failed to update agent exception resolution",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		if err = dberror.CheckAffected(
			int64(len(rows)),
			"AgentException",
			req.ID.String(),
		); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}
