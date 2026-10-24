package agentsuggestionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion/agentsuggestiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentsuggestionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var suggestionErrors = dberror.Mapping{Entity: "Agent suggestion"}

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
		if err := agentsuggestiondb.Suggestions.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to record an agent suggestion", zap.Error(err))

			return nil, fmt.Errorf("record agent suggestion: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) Settle(
	ctx context.Context,
	req *agentsuggestionports.SettleAgentSuggestionRow,
) (*agentsuggestion.Suggestion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentsuggestion.Suggestion, error) {
		s := agentsuggestiondb.Suggestions
		rows, err := s.Where(
			s.ID.Eq(req.ID),
			s.UserID.Eq(req.UserID),
			s.Outcome.Eq(agentsuggestion.OutcomeShown),
		).UpdateReturning(ctx, r.db.Keel(ctx), s.Outcome.To(req.Outcome))
		if err != nil {
			return nil, fmt.Errorf("settle agent suggestion: %w", err)
		}
		if len(rows) > 0 {
			return rows[0], nil
		}

		settled, err := s.Where(s.ID.Eq(req.ID), s.UserID.Eq(req.UserID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, suggestionErrors.Map(err, req.ID.String())
		}

		return settled, nil
	})
}
