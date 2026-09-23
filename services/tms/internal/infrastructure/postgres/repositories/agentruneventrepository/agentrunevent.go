package agentruneventrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.AgentRunEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentrunevent-repository"),
	}
}

// Append writes a batch of events.
//
// There is no conflict clause. A duplicated sequence is a writer that lost
// count, and the unique index refusing it is the correct outcome: silently
// absorbing it would leave a trajectory that reads plausibly and is wrong.
func (r *repository) Append(
	ctx context.Context,
	req repositories.AppendAgentRunEventsRequest,
) error {
	if len(req.Events) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).NewInsert().
		Model(&req.Events).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to append agent run events",
			zap.Int("events", len(req.Events)),
			zap.Error(err),
		)

		return fmt.Errorf("append agent run events: %w", err)
	}

	return nil
}

// NextSequence is where a writer resumes counting.
//
// A run that is retried starts a fresh writer with no memory of the numbers the
// last attempt used, so the count is read back from what is already stored
// rather than started again at one, which would collide.
func (r *repository) NextSequence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ownerKind string,
	ownerID pulid.ID,
) (int, error) {
	cols := buncolgen.AgentRunEventColumns

	var highest int
	err := r.db.DBForContext(ctx).NewSelect().
		Model((*agent.AgentRunEvent)(nil)).
		ColumnExpr("COALESCE(MAX(?), 0)", bun.Ident(cols.Sequence.String())).
		Apply(buncolgen.AgentRunEventApplyTenant(tenantInfo)).
		Where(cols.OwnerKind.Eq(), ownerKind).
		Where(cols.OwnerID.Eq(), ownerID).
		Scan(ctx, &highest)
	if err != nil {
		r.l.Error("failed to read the next event sequence",
			zap.String("owner", ownerID.String()),
			zap.Error(err),
		)

		return 0, fmt.Errorf("read next agent run event sequence: %w", err)
	}

	return highest + 1, nil
}

func (r *repository) List(
	ctx context.Context,
	req repositories.ListAgentRunEventsRequest,
) ([]*agent.AgentRunEvent, error) {
	cols := buncolgen.AgentRunEventColumns
	events := make([]*agent.AgentRunEvent, 0, 32)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(&events).
		Apply(buncolgen.AgentRunEventApplyTenant(req.TenantInfo)).
		Where(cols.OwnerKind.Eq(), req.OwnerKind).
		Where(cols.OwnerID.Eq(), req.OwnerID).
		Order(cols.Sequence.OrderAsc())

	if req.After > 0 {
		q = q.Where(cols.Sequence.Gt(), req.After)
	}

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list agent run events",
			zap.String("owner", req.OwnerID.String()),
			zap.Error(err),
		)

		return nil, fmt.Errorf("list agent run events: %w", err)
	}

	return events, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentRunEventConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRunEvent], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*agent.AgentRunEvent)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AgentRunEventTable.Alias,
					req.Filter,
					(*agent.AgentRunEvent)(nil),
				)

				return sq.Apply(buncolgen.AgentRunEventApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count agent run events", zap.Error(err))

			return nil, fmt.Errorf("count agent run events: %w", err)
		}

		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*agent.AgentRunEvent]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*agent.AgentRunEvent) *bun.SelectQuery {
			q := dba.NewSelect().Model(entities)
			if len(req.Columns) > 0 {
				q = q.Column(req.Columns...)
			}

			return q
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.AgentRunEventTable.Alias,
				req.Filter,
				req.Cursor,
				(*agent.AgentRunEvent)(nil),
			)
		},
	})
	if err != nil {
		log.Error("failed to list agent run events", zap.Error(err))

		return nil, fmt.Errorf("list agent run events: %w", err)
	}

	return result, nil
}

// Prune drops a batch of old events. Deliberately not tenant-scoped: the sweep
// runs for the whole installation.
func (r *repository) Prune(
	ctx context.Context,
	req repositories.PruneAgentRunEventsRequest,
) (int, error) {
	cols := buncolgen.AgentRunEventColumns

	dba := r.db.DBForContext(ctx)
	doomed := dba.NewSelect().
		Model((*agent.AgentRunEvent)(nil)).
		Column(cols.ID.String()).
		Where(cols.CreatedAt.Lt(), req.Before).
		Limit(req.Limit)

	result, err := dba.NewDelete().
		Model((*agent.AgentRunEvent)(nil)).
		Where(cols.ID.In(), doomed).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to prune agent run events", zap.Error(err))

		return 0, fmt.Errorf("prune agent run events: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune agent run events: %w", err)
	}

	return int(affected), nil
}
