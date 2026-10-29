package agentevaluationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation/agentevaluationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentevaluationports"
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

func New(p Params) agentevaluationports.AgentEvaluationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentevaluation-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentevaluation.Evaluation,
) (*agentevaluation.Evaluation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentevaluation.Evaluation, error) {
		if err := agentevaluationdb.Evaluations.Create(ctx, r.db.Keel(ctx), entity); err != nil {
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
	entity *agentevaluation.Evaluation,
) (*agentevaluation.Evaluation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentevaluation.Evaluation, error) {
		e := agentevaluationdb.Evaluations
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
	req agentevaluationports.GetAgentEvaluationByIDRequest,
) (*agentevaluation.Evaluation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentevaluation.Evaluation, error) {
		entity, err := agentevaluationdb.Evaluations.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, evaluationErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentevaluationports.ListAgentEvaluationConnectionRequest,
) (*pagination.CursorListResult[*agentevaluation.Evaluation], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agentevaluation.Evaluation], error) {
			e := agentevaluationdb.Evaluations
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				e.Where(keel.When(req.SuiteRunID.IsNotNil(), e.SuiteRunID.Eq(req.SuiteRunID))),
				&listquery.Request[agentevaluation.Evaluation]{
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

func (r *repository) CreateMany(ctx context.Context, entities []*agentevaluation.Evaluation) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entities) == 0 {
			return nil
		}

		if err := agentevaluationdb.Evaluations.CreateMany(
			ctx,
			r.db.Keel(ctx),
			entities,
		); err != nil {
			r.l.Error("failed to create suite evaluations", zap.Error(err))

			return fmt.Errorf("create suite evaluations: %w", err)
		}

		return nil
	})
}

func (r *repository) ListBySuiteRun(
	ctx context.Context,
	req *agentevaluationports.ListSuiteEvaluationsRequest,
) ([]*agentevaluation.Evaluation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentevaluation.Evaluation, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxSuitePage {
			limit = maxSuitePage
		}

		e := agentevaluationdb.Evaluations
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
	req *agentevaluationports.SkipPendingSuiteEvaluationsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		e := agentevaluationdb.Evaluations
		n, err := e.Where(
			e.SuiteRunID.Eq(req.SuiteRunID),
			e.Status.Eq(agentevaluation.EvaluationStatusPending),
		).Update(
			ctx,
			r.db.Keel(ctx),
			e.Status.To(agentevaluation.EvaluationStatusSkipped),
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
