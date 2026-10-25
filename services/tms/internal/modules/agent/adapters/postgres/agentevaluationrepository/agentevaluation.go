package agentevaluationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxSuitePage = 200

var evaluationErrors = dberror.Mapping{Entity: "AgentEvaluation"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AgentEvaluationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentevaluation-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.Evaluation,
) (*agent.Evaluation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Evaluation, error) {
		if err := agentdb.Evaluations.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent evaluation", zap.Error(err))

			return nil, fmt.Errorf("create agent evaluation: %w", err)
		}

		return entity, nil
	})
}

// Update writes the whole outcome under the version, since one workflow
// owns an evaluation from start to finish and nothing else edits it.
func (r *repository) Update(
	ctx context.Context,
	entity *agent.Evaluation,
) (*agent.Evaluation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Evaluation, error) {
		e := agentdb.Evaluations
		err := e.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			e.Status,
			e.Input,
			e.DefinitionVersion,
			e.PromptVersion,
			e.Model,
			e.ProviderID,
			e.Reply,
			e.Actions,
			e.Comparison,
			e.OriginalProposals,
			e.ToolCallsUsed,
			e.Checks,
			e.Judge,
			e.CaseScore,
			e.Fingerprint,
			e.WorkflowID,
			e.ErrorMessage,
			e.StartedAt,
			e.CompletedAt,
		)
		if err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, evaluationErrors.Map(err, entity.ID.String())
			}

			return nil, fmt.Errorf("update agent evaluation: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAgentEvaluationByIDRequest,
) (*agent.Evaluation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.Evaluation, error) {
		entity, err := agentdb.Evaluations.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, evaluationErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentEvaluationConnectionRequest,
) (*pagination.CursorListResult[*agent.Evaluation], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.Evaluation], error) {
			e := agentdb.Evaluations
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				e.Where(keel.When(req.SuiteRunID.IsNotNil(), e.SuiteRunID.Eq(req.SuiteRunID))),
				&listquery.Request[agent.Evaluation]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list agent evaluations", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) CreateMany(ctx context.Context, entities []*agent.Evaluation) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entities) == 0 {
			return nil
		}

		if err := agentdb.Evaluations.CreateMany(ctx, r.db.Keel(ctx), entities); err != nil {
			r.l.Error("failed to create suite evaluations", zap.Error(err))

			return fmt.Errorf("create suite evaluations: %w", err)
		}

		return nil
	})
}

func (r *repository) ListBySuiteRun(
	ctx context.Context,
	req agentports.ListSuiteEvaluationsRequest,
) ([]*agent.Evaluation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Evaluation, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxSuitePage {
			limit = maxSuitePage
		}

		e := agentdb.Evaluations
		entities, err := e.Where(e.SuiteRunID.Eq(req.SuiteRunID), e.SuiteOrdinal.Gt(int64(req.AfterOrdinal))).
			Order(e.SuiteOrdinal.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list suite evaluations: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) SkipPendingBySuiteRun(
	ctx context.Context,
	req agentports.SkipPendingSuiteEvaluationsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		e := agentdb.Evaluations
		n, err := e.Where(
			e.SuiteRunID.Eq(req.SuiteRunID),
			e.Status.Eq(agent.EvaluationStatusPending),
		).Update(
			ctx,
			r.db.Keel(ctx),
			e.Status.To(agent.EvaluationStatusSkipped),
			e.ErrorMessage.To(req.Reason),
			e.CompletedAt.To(req.At),
			e.UpdatedAt.To(req.At),
		)
		if err != nil {
			return 0, fmt.Errorf("skip pending suite evaluations: %w", err)
		}

		return int(n), nil
	})
}
