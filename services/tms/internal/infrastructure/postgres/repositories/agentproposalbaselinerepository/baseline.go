package agentproposalbaselinerepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// DefaultPurgeLimit bounds one sweep of orphaned baselines.
const DefaultPurgeLimit = 1000

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.AgentProposalBaselineRepository {
	return &repository{db: p.DB}
}

var conflictTarget = "CONFLICT (" +
	strings.Join(buncolgen.ProposalBaselineTable.PrimaryKey, ", ") + ") DO NOTHING"

func (r *repository) Create(ctx context.Context, baseline *agent.ProposalBaseline) error {
	if _, err := r.db.DBForContext(ctx).NewInsert().
		Model(baseline).
		On(conflictTarget).
		Exec(ctx); err != nil {
		return fmt.Errorf("keep the proposal baseline: %w", err)
	}

	return nil
}

func (r *repository) GetByProposal(
	ctx context.Context,
	req repositories.GetProposalBaselineRequest,
) (*agent.ProposalBaseline, error) {
	baseline := new(agent.ProposalBaseline)
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(baseline).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return scope(q, req.TenantInfo).
				Where(buncolgen.ProposalBaselineColumns.ProposalID.Eq(), req.ProposalID)
		}).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "ProposalBaseline")
	}

	return baseline, nil
}

func (r *repository) ListByProposals(
	ctx context.Context,
	req repositories.ListProposalBaselinesRequest,
) ([]*agent.ProposalBaseline, error) {
	rows := make([]*agent.ProposalBaseline, 0, len(req.ProposalIDs))
	if len(req.ProposalIDs) == 0 {
		return rows, nil
	}

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return scope(q, req.TenantInfo).
				Where(buncolgen.ProposalBaselineColumns.ProposalID.In(), bun.List(req.ProposalIDs))
		}).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list proposal baselines: %w", err)
	}

	return rows, nil
}

func (r *repository) PurgeOrphans(
	ctx context.Context,
	req repositories.PurgeOrphanProposalBaselinesRequest,
) (int, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultPurgeLimit
	}

	result, err := purgeQuery(r.db.DBForContext(ctx), req.Before, limit).Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("purge orphaned proposal baselines: %w", err)
	}

	purged, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count purged proposal baselines: %w", err)
	}

	return int(purged), nil
}

// purgeQuery deletes baselines older than before that no proposal row names.
// A proposal is filed after its baseline is kept, so only an old baseline
// can be known to be an orphan: its filing was retried under another id, or
// never happened.
func purgeQuery(db bun.IDB, before int64, limit int) *bun.DeleteQuery {
	cols := buncolgen.ProposalBaselineColumns
	proposal := buncolgen.AgentProposalColumns

	filed := db.NewSelect().
		Model((*agent.AgentProposal)(nil)).
		ColumnExpr("1").
		Where(proposal.ID.EqColumn(cols.ProposalID)).
		Where(proposal.OrganizationID.EqColumn(cols.OrganizationID)).
		Where(proposal.BusinessUnitID.EqColumn(cols.BusinessUnitID))

	orphans := db.NewSelect().
		Model((*agent.ProposalBaseline)(nil)).
		Column(cols.ProposalID.String(), cols.OrganizationID.String(), cols.BusinessUnitID.String()).
		Where(cols.CreatedAt.Lt(), before).
		Where("NOT EXISTS (?)", filed).
		OrderExpr(cols.CreatedAt.OrderAsc()).
		Limit(limit)

	return db.NewDelete().
		Model((*agent.ProposalBaseline)(nil)).
		Where(
			buncolgen.Expr("({0}, {1}, {2}) IN (?)",
				cols.ProposalID, cols.OrganizationID, cols.BusinessUnitID),
			orphans,
		)
}

func scope(q *bun.SelectQuery, tenant pagination.TenantInfo) *bun.SelectQuery {
	return buncolgen.ProposalBaselineScopeTenant(q, tenant)
}
