package agentshadowrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.AgentShadowRepository {
	return &repository{db: p.DB}
}

type shadowProposalRow struct {
	TargetID       string                `bun:"target_id"`
	CreatedAt      int64                 `bun:"created_at"`
	Status         agent.ProposalStatus  `bun:"status"`
	ExecutionError string                `bun:"execution_error"`
	Simulation     *agent.ToolSimulation `bun:"simulation,type:jsonb"`
}

func (r *repository) ListShadowProposals(
	ctx context.Context,
	req *repositories.ListShadowProposalsRequest,
) ([]agentshadow.Proposal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentshadow.Proposal, error) {
		cols := buncolgen.AgentProposalColumns
		runCols := buncolgen.AgentRunColumns
		rows := make([]shadowProposalRow, 0, req.Limit)

		err := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("agent_proposals AS ap").
			ColumnExpr(cols.TargetID.Qualified()).
			ColumnExpr(cols.CreatedAt.Qualified()).
			ColumnExpr(cols.Status.Qualified()).
			ColumnExpr(cols.ExecutionError.Qualified()).
			ColumnExpr(cols.Simulation.Qualified()).
			Join("JOIN agent_runs AS ar ON "+runCols.ID.Qualified()+" = "+cols.RunID.Qualified()+
				" AND "+runCols.OrganizationID.Qualified()+" = "+cols.OrganizationID.Qualified()+
				" AND "+runCols.BusinessUnitID.Qualified()+" = "+cols.BusinessUnitID.Qualified()).
			Where(runCols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(runCols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Where(runCols.AgentDefinitionID.Eq(), req.AgentDefinitionID).
			Where(runCols.Status.Eq(), agent.RunStatusShadowCompleted).
			Where(runCols.CreatedAt.Gte(), req.Since).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			Limit(req.Limit).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list shadow proposals: %w", err)
		}

		proposals := make([]agentshadow.Proposal, 0, len(rows))
		for idx := range rows {
			proposals = append(proposals, rows[idx].proposal())
		}
		return proposals, nil
	})
}

func (row *shadowProposalRow) proposal() agentshadow.Proposal {
	proposal := agentshadow.Proposal{
		TargetID:  row.TargetID,
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

type personChangeRow struct {
	ResourceID string         `bun:"resource_id"`
	Timestamp  int64          `bun:"timestamp"`
	Changes    map[string]any `bun:"changes,type:jsonb"`
}

func (r *repository) ListPersonChanges(
	ctx context.Context,
	req *repositories.ListPersonChangesRequest,
) ([]agentshadow.PersonChange, error) {
	if len(req.ResourceIDs) == 0 {
		return []agentshadow.PersonChange{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentshadow.PersonChange, error) {
		cols := buncolgen.EntryColumns
		rows := make([]personChangeRow, 0, len(req.ResourceIDs))

		err := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("audit_entries AS ae").
			ColumnExpr(cols.ResourceID.Qualified()).
			ColumnExpr(cols.Timestamp.Qualified()).
			ColumnExpr(cols.Changes.Qualified()).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Where(cols.ResourceID.In(), bun.List(req.ResourceIDs)).
			Where(cols.Timestamp.Between(), req.Since, req.Until).
			Where(cols.PrincipalType.Eq(), authctx.PrincipalTypeUser).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list person changes: %w", err)
		}

		changes := make([]agentshadow.PersonChange, 0, len(rows))
		for idx := range rows {
			fields := make([]string, 0, len(rows[idx].Changes))
			for field := range rows[idx].Changes {
				fields = append(fields, field)
			}
			changes = append(changes, agentshadow.PersonChange{
				ResourceID: rows[idx].ResourceID,
				Timestamp:  rows[idx].Timestamp,
				Fields:     fields,
			})
		}
		return changes, nil
	})
}
