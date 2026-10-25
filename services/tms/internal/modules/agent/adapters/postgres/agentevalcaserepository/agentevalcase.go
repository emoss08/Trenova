package agentevalcaserepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality/agentqualitydb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit  = 500
	maxListLimit      = 2000
	defaultBatchLimit = 200
	maxBatchLimit     = 1000
	entityName        = "AgentEvalCase"
)

var evalCaseErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentqualityports.AgentEvalCaseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentevalcase-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentquality.EvalCase,
) (*agentquality.EvalCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentquality.EvalCase, error) {
		if err := agentqualitydb.EvalCases.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, fmt.Errorf("create agent eval case: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentquality.EvalCase,
) (*agentquality.EvalCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentquality.EvalCase, error) {
		c := agentqualitydb.EvalCases
		err := c.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			c.Title,
			c.Status,
			c.Input,
			c.History,
			c.PageContext,
			c.Mentions,
			c.SubjectType,
			c.SubjectID,
			c.HeldTools,
			c.ToolFixtures,
			c.Expected,
			c.Rubric,
			c.Redaction,
			c.ContentHash,
			c.ExpiresAt,
		)
		if err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, evalCaseErrors.Map(err, entity.ID.String())
			}

			return nil, fmt.Errorf("update agent eval case: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentqualityports.GetAgentEvalCaseByIDRequest,
) (*agentquality.EvalCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentquality.EvalCase, error) {
		entity, err := agentqualitydb.EvalCases.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, evalCaseErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByContent(
	ctx context.Context,
	req agentqualityports.GetAgentEvalCaseByContentRequest,
) (*agentquality.EvalCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentquality.EvalCase, error) {
		c := agentqualitydb.EvalCases
		entity, err := c.Where(
			c.AgentDefinitionID.Eq(req.AgentDefinitionID),
			c.ContentHash.Eq(req.ContentHash),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, evalCaseErrors.Map(err, req.ContentHash)
		}

		return entity, nil
	})
}

func (r *repository) GetByProposal(
	ctx context.Context,
	req agentqualityports.GetAgentEvalCaseByProposalRequest,
) (*agentquality.EvalCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentquality.EvalCase, error) {
		c := agentqualitydb.EvalCases
		entity, err := c.Where(c.SourceProposalID.Eq(req.ProposalID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, evalCaseErrors.Map(err, req.ProposalID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentqualityports.ListAgentEvalCaseConnectionRequest,
) (*pagination.CursorListResult[*agentquality.EvalCase], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agentquality.EvalCase], error) {
			c := agentqualitydb.EvalCases
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				c.Where(
					keel.When(
						req.AgentDefinitionID.IsNotNil(),
						c.AgentDefinitionID.Eq(req.AgentDefinitionID),
					),
					keel.When(len(req.Statuses) > 0, c.Status.In(req.Statuses...)),
					keel.When(len(req.Sources) > 0, c.Source.In(req.Sources...)),
				),
				&listquery.Request[agentquality.EvalCase]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list agent eval cases", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListByAgent(
	ctx context.Context,
	req agentqualityports.ListAgentEvalCasesRequest,
) ([]*agentquality.EvalCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentquality.EvalCase, error) {
		c := agentqualitydb.EvalCases
		entities, err := c.Where(
			c.AgentDefinitionID.Eq(req.AgentDefinitionID),
			keel.When(len(req.Statuses) > 0, c.Status.In(req.Statuses...)),
		).
			Order(c.CreatedAt.Asc(), c.ID.Asc()).
			Limit(clamp(req.Limit, defaultListLimit, maxListLimit)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent eval cases: %w", err)
		}

		return entities, nil
	})
}

func capturedDecisions() []agent.DecisionType {
	return []agent.DecisionType{
		agent.DecisionAccepted,
		agent.DecisionModified,
		agent.DecisionRejected,
	}
}

func (r *repository) ListCaptureCandidates(
	ctx context.Context,
	req agentqualityports.ListEvalCaseCaptureCandidatesRequest,
) ([]agentqualityports.EvalCaseCaptureCandidate, error) {
	ctx = dbscope.WithSystem(ctx, "list evaluation case candidates across every organization")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentqualityports.EvalCaseCaptureCandidate, error) {
			d := agentdb.AgentDecisions
			p := agentdb.AgentProposals
			run := agentdb.AgentRuns
			captured := agentqualitydb.EvalCases.As("captured")

			rows, err := keel.Into[agentqualityports.EvalCaseCaptureCandidate](
				ctx,
				r.db.Keel(ctx),
				d.AllTenants().
					Pick(
						d.OrganizationID,
						d.BusinessUnitID,
						d.ProposalID,
						d.ID,
						d.DecidedByUserID,
						run.AgentDefinitionID,
						d.CreatedAt,
					).
					Join(p,
						p.ID.EqCol(d.ProposalID),
						p.OrganizationID.EqCol(d.OrganizationID),
						p.BusinessUnitID.EqCol(d.BusinessUnitID),
					).
					Join(run,
						run.ID.EqCol(p.RunID),
						run.OrganizationID.EqCol(p.OrganizationID),
						run.BusinessUnitID.EqCol(p.BusinessUnitID),
					).
					Where(
						d.ProposalID.IsNotNull(),
						d.Decision.In(capturedDecisions()...),
						d.CreatedAt.Gte(req.Since),
						run.AgentDefinitionID.IsNotNull(),
						keel.NotExists(captured.AllTenants().Where(
							captured.OrganizationID.EqCol(d.OrganizationID),
							captured.BusinessUnitID.EqCol(d.BusinessUnitID),
							captured.SourceProposalID.EqCol(d.ProposalID),
						)),
					).
					Order(d.CreatedAt.Asc(), d.ID.Asc()).
					Limit(clamp(req.Limit, defaultBatchLimit, maxBatchLimit)),
			)
			if err != nil {
				return nil, fmt.Errorf("list eval case capture candidates: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) PurgeExpired(
	ctx context.Context,
	req agentqualityports.PurgeExpiredEvalCasesRequest,
) (int, error) {
	if req.AllTenants {
		ctx = dbscope.WithSystem(ctx, "purge expired evaluation cases across every organization")
	}
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		due := agentqualitydb.EvalCases.As("due")
		keys := due.Pick(due.ID).
			Where(keel.Or(
				keel.And(due.ExpiresAt.IsNotNull(), due.ExpiresAt.Lte(req.Now)),
				keel.When(req.CreatedBefore > 0, due.CreatedAt.Lt(req.CreatedBefore)),
			)).
			Order(due.CreatedAt.Asc()).
			Limit(clamp(req.Limit, defaultBatchLimit, maxBatchLimit))

		return r.deleteKeys(ctx, keys, req.AllTenants)
	})
}

func (r *repository) PurgeOrphaned(
	ctx context.Context,
	req agentqualityports.PurgeOrphanedEvalCasesRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "purge orphaned evaluation cases across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		orphan := agentqualitydb.EvalCases.As("orphan")
		thread := conversationdb.Threads
		keys := orphan.Pick(orphan.ID).
			Where(
				orphan.SourceThreadID.IsNotNull(),
				keel.NotExists(thread.AllTenants().Where(
					thread.ID.EqCol(orphan.SourceThreadID),
					thread.OrganizationID.EqCol(orphan.OrganizationID),
					thread.BusinessUnitID.EqCol(orphan.BusinessUnitID),
				)),
			).
			Order(orphan.CreatedAt.Asc()).
			Limit(clamp(req.Limit, defaultBatchLimit, maxBatchLimit))

		return r.deleteKeys(ctx, keys, true)
	})
}

func (r *repository) deleteKeys(
	ctx context.Context,
	keys *keel.Query[agentquality.EvalCase],
	allTenants bool,
) (int, error) {
	c := agentqualitydb.EvalCases
	q := c.Query()
	if allTenants {
		q = q.AllTenants()
		keys = keys.AllTenants()
	}

	n, err := q.Where(c.ID.InQuery(keys)).Delete(ctx, r.db.Keel(ctx))
	if err != nil {
		return 0, fmt.Errorf("purge agent eval cases: %w", err)
	}

	return int(n), nil
}

func clamp(value, fallback, ceiling int) int {
	return intutils.Clamp(intutils.WithDefault(value, fallback), 1, ceiling)
}

func (r *repository) ListAgentsWithActiveCases(
	ctx context.Context,
	_ agentqualityports.ListAgentsWithActiveEvalCasesRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		c := agentqualitydb.EvalCases
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), c.
			Group(c.AgentDefinitionID).
			Pick(c.AgentDefinitionID).
			Where(c.Status.Eq(agentquality.CaseStatusActive)).
			Order(c.AgentDefinitionID.Asc()).
			Limit(maxListLimit))
		if err != nil {
			return nil, fmt.Errorf("list agents with active eval cases: %w", err)
		}
		if ids == nil {
			ids = []pulid.ID{}
		}

		return ids, nil
	})
}

func (r *repository) ListSamplingCases(
	ctx context.Context,
	req agentqualityports.ListEvalCaseSamplingRequest,
) ([]agentquality.SamplingCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentquality.SamplingCase, error) {
		c := agentqualitydb.EvalCases
		ev := agentdb.Evaluations

		lastFailure := keel.Scalar(ev.
			Pick(keel.Raw(
				"CASE WHEN ? = ? OR (? ->> 'passed') = 'false' THEN COALESCE(?, ?) ELSE 0 END",
				ev.Status,
				string(agent.EvaluationStatusFailed),
				ev.Checks,
				ev.CompletedAt,
				ev.UpdatedAt,
			)).
			Where(
				ev.EvalCaseID.EqCol(c.ID),
				ev.Status.In(agent.EvaluationStatusCompleted, agent.EvaluationStatusFailed),
			).
			Order(ev.CreatedAt.Desc(), ev.ID.Desc()).
			Limit(1))

		cases, err := keel.Into[agentquality.SamplingCase](ctx, r.db.Keel(ctx), c.
			Pick(
				c.ID,
				c.Source,
				c.Version,
				c.Trigger,
				keel.Raw("COALESCE(?, '')", c.SubjectType),
				c.SubjectID,
				keel.Raw("COALESCE(?, 0)", lastFailure),
			).
			Where(
				c.AgentDefinitionID.Eq(req.AgentDefinitionID),
				c.Status.Eq(agentquality.CaseStatusActive),
			).
			Order(c.ID.Asc()).
			Limit(maxListLimit))
		if err != nil {
			return nil, fmt.Errorf("list eval cases to sample: %w", err)
		}
		if cases == nil {
			cases = []agentquality.SamplingCase{}
		}

		return cases, nil
	})
}
