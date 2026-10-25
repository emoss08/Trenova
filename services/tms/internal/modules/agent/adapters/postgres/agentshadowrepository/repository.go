package agentshadowrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentshadowports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/pkg/authctx"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) agentshadowports.AgentShadowRepository {
	return &repository{db: p.DB}
}

func (r *repository) ListShadowProposals(
	ctx context.Context,
	req *agentshadowports.ListShadowProposalsRequest,
) ([]agentshadow.Proposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentshadow.Proposal, error) {
		p := agentdb.AgentProposals
		run := p.Run()
		rows, err := p.Pick(p.TargetID, p.CreatedAt, p.Status, p.ExecutionError, p.Simulation).
			Where(
				run.AgentDefinitionID.Eq(req.AgentDefinitionID),
				run.Status.Eq(agent.RunStatusShadowCompleted),
				run.CreatedAt.Gte(req.Since),
			).
			Order(p.CreatedAt.Desc()).
			Limit(req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list shadow proposals: %w", err)
		}

		proposals := make([]agentshadow.Proposal, 0, len(rows))
		for _, row := range rows {
			proposals = append(proposals, shadowProposal(row))
		}
		return proposals, nil
	})
}

func shadowProposal(row *agent.AgentProposal) agentshadow.Proposal {
	proposal := agentshadow.Proposal{
		TargetID:  row.TargetID.String(),
		CreatedAt: row.CreatedAt,
		Failed:    row.Status == agent.ProposalStatusExecutionFailed || row.ExecutionError != "",
	}
	if row.Simulation != nil {
		proposal.Fields = make([]string, 0, len(row.Simulation.Changes))
		for _, change := range row.Simulation.Changes {
			proposal.Fields = append(proposal.Fields, change.Field)
		}
	}
	return proposal
}

func (r *repository) ListPersonChanges(
	ctx context.Context,
	req *agentshadowports.ListPersonChangesRequest,
) ([]agentshadow.PersonChange, error) {
	if len(req.ResourceIDs) == 0 {
		return []agentshadow.PersonChange{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentshadow.PersonChange, error) {
		ae := auditdb.Entries
		rows, err := ae.Pick(ae.ResourceID, ae.Timestamp, ae.Changes).
			Where(
				ae.ResourceID.In(req.ResourceIDs...),
				ae.Timestamp.Between(req.Since, req.Until),
				ae.PrincipalType.Eq(authctx.PrincipalTypeUser),
			).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list person changes: %w", err)
		}

		changes := make([]agentshadow.PersonChange, 0, len(rows))
		for _, row := range rows {
			fields := make([]string, 0, len(row.Changes))
			for field := range row.Changes {
				fields = append(fields, field)
			}
			changes = append(changes, agentshadow.PersonChange{
				ResourceID: row.ResourceID,
				Timestamp:  row.Timestamp,
				Fields:     fields,
			})
		}
		return changes, nil
	})
}
