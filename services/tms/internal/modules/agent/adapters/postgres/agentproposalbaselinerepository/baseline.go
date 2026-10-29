package agentproposalbaselinerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

// DefaultPurgeLimit bounds one sweep of orphaned baselines.
const DefaultPurgeLimit = 1000

var baselineErrors = dberror.Mapping{Entity: "ProposalBaseline"}

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) agentports.AgentProposalBaselineRepository {
	return &repository{db: p.DB}
}

func (r *repository) Create(ctx context.Context, baseline *agentpreview.ProposalBaseline) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		b := agentdb.ProposalBaselines
		if _, err := b.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*agentpreview.ProposalBaseline{baseline},
			b.ProposalID,
			b.OrganizationID,
			b.BusinessUnitID,
		); err != nil {
			return fmt.Errorf("keep the proposal baseline: %w", err)
		}

		return nil
	})
}

func (r *repository) GetByProposal(
	ctx context.Context,
	req agentports.GetProposalBaselineRequest,
) (*agentpreview.ProposalBaseline, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentpreview.ProposalBaseline, error) {
		baseline, err := agentdb.ProposalBaselines.Get(ctx, r.db.Keel(ctx), req.ProposalID)
		if err != nil {
			return nil, baselineErrors.Map(err, req.ProposalID.String())
		}

		return baseline, nil
	})
}

func (r *repository) ListByProposals(
	ctx context.Context,
	req agentports.ListProposalBaselinesRequest,
) ([]*agentpreview.ProposalBaseline, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentpreview.ProposalBaseline, error) {
			if len(req.ProposalIDs) == 0 {
				return []*agentpreview.ProposalBaseline{}, nil
			}

			b := agentdb.ProposalBaselines
			rows, err := b.Where(b.ProposalID.In(req.ProposalIDs...)).
				Limit(len(req.ProposalIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list proposal baselines: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) PurgeOrphans(
	ctx context.Context,
	req agentports.PurgeOrphanProposalBaselinesRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "purge orphaned proposal baselines across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = DefaultPurgeLimit
		}

		b := agentdb.ProposalBaselines
		purged, err := b.AllTenants().
			Where(b.ProposalID.InQuery(orphans(req.Before, limit))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("purge orphaned proposal baselines: %w", err)
		}

		return int(purged), nil
	})
}

// orphans are the oldest baselines older than before that no proposal row
// names. A proposal is filed after its baseline is kept, so only an old
// baseline can be known to be an orphan: its filing was retried under
// another id, or never happened.
func orphans(before int64, limit int) *keel.Query[*agentpreview.ProposalBaseline] {
	orphan := agentdb.ProposalBaselines.As("orphan")
	p := agentdb.AgentProposals

	return orphan.AllTenants().
		Pick(orphan.ProposalID).
		Where(
			orphan.CreatedAt.Lt(before),
			keel.NotExists(p.AllTenants().Where(
				p.ID.EqCol(orphan.ProposalID),
				p.OrganizationID.EqCol(orphan.OrganizationID),
				p.BusinessUnitID.EqCol(orphan.BusinessUnitID),
			)),
		).
		Order(orphan.CreatedAt.Asc()).
		Limit(limit)
}
