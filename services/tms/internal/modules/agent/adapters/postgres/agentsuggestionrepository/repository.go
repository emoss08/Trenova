package agentsuggestionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentsuggestionports"
	"github.com/emoss08/trenova/pkg/buncolgen/agentsuggestioncol"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) agentsuggestionports.AgentSuggestionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-suggestion-repository"),
	}
}

func (r *repository) Insert(
	ctx context.Context,
	entity *agentsuggestion.Suggestion,
) (*agentsuggestion.Suggestion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentsuggestion.Suggestion, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to record an agent suggestion", zap.Error(err))

			return nil, fmt.Errorf("record agent suggestion: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) Settle(
	ctx context.Context,
	req agentsuggestionports.SettleAgentSuggestionRow,
) (*agentsuggestion.Suggestion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentsuggestion.Suggestion, error) {
		cols := agentsuggestioncol.SuggestionColumns
		entity := new(agentsuggestion.Suggestion)

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Set(cols.Outcome.Set(), req.Outcome).
			Set(cols.Version.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return agentsuggestioncol.SuggestionScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.UserID.Eq(), req.UserID).
					Where(cols.Outcome.Eq(), agentsuggestion.OutcomeShown)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("settle agent suggestion: %w", err)
		}
		if affected, _ := res.RowsAffected(); affected > 0 {
			return entity, nil
		}

		settled := new(agentsuggestion.Suggestion)
		err = r.db.DBForContext(ctx).
			NewSelect().
			Model(settled).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentsuggestioncol.SuggestionScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.UserID.Eq(), req.UserID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Agent suggestion")
		}

		return settled, nil
	})
}
