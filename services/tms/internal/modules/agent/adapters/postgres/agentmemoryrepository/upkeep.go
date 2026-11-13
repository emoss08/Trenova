package agentmemoryrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory/agentmemorydb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/dberror"
)

const maxUpkeepLimit = 50

func (r *repository) Confirm(
	ctx context.Context,
	req *agentmemoryports.ConfirmAgentMemoryQuery,
) (*agentmemory.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentmemory.Memory, error) {
		m := agentmemorydb.Memories
		n, err := m.Where(
			m.ID.Eq(req.ID),
			m.Version.Eq(req.Version),
			m.Status.Eq(agentmemory.MemoryStatusActive),
		).Update(
			ctx,
			r.db.Keel(ctx),
			setID(m.ConfirmedByUserID, req.ByUserID),
			m.ConfirmedAt.To(req.At),
			m.UpdatedAt.To(req.At),
		)
		if err != nil {
			return nil, fmt.Errorf("confirm agent memory: %w", err)
		}
		if err = dberror.CheckAffected(n, memoryEntity, req.ID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(
			ctx,
			agentmemoryports.GetAgentMemoryByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
		)
	})
}

func (r *repository) ListUpkeep(
	ctx context.Context,
	req *agentmemoryports.ListAgentMemoryUpkeepRequest,
) ([]*agentmemory.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentmemory.Memory, error) {
		m := agentmemorydb.Memories
		limit := req.Limit
		if limit <= 0 || limit > maxUpkeepLimit {
			limit = maxUpkeepLimit
		}

		rows, err := m.Where(
			m.Status.Eq(agentmemory.MemoryStatusActive),
			m.Scope.Ne(agentmemory.MemoryScopeUser),
			keel.Or(
				keel.And(
					m.Kind.Eq(agentmemory.MemoryKindFact),
					m.ExpiresAt.IsNull(),
					vouchedBefore(req.Window.AgingBefore),
				),
				keel.And(m.UseCount.Eq(0), vouchedBefore(req.Window.UnusedBefore)),
			),
		).
			Order(m.CreatedAt.Asc(), m.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent memories worth another look: %w", err)
		}

		return rows, nil
	})
}

// vouchedBefore matches a memory nobody has stood behind since the time
// given: confirmed before it, or never confirmed and recorded before it.
func vouchedBefore(at int64) keel.Pred {
	m := agentmemorydb.Memories

	return keel.Or(
		m.ConfirmedAt.Lt(at),
		keel.And(m.ConfirmedAt.IsNull(), m.CreatedAt.Lt(at)),
	)
}

func (r *repository) ListOutcomes(
	ctx context.Context,
	req *agentmemoryports.ListAgentMemoryOutcomesRequest,
) ([]agentmemoryports.MemoryOutcome, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemoryOutcome, error) {
			m := agentmemorydb.Memories
			trigger := keel.StringOf(m.Evidence.Text("signals", "0")).Coalesce("")

			rows, err := keel.Into[agentmemoryports.MemoryOutcome](ctx, r.db.Keel(ctx), m.
				Where(m.CreatedAt.Gte(req.Since)).
				Group(m.Source, trigger, m.Status).
				Pick(
					m.Source,
					trigger,
					m.Status,
					keel.Count(),
					keel.Count().Filter(m.UseCount.Gt(0)),
					keel.Count().Filter(keel.And(
						m.Status.Eq(agentmemory.MemoryStatusDismissed),
						m.RetiredByUserID.IsNull(),
					)),
				))
			if err != nil {
				return nil, fmt.Errorf("count what became of agent memories: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) LapseOffers(
	ctx context.Context,
	req *agentmemoryports.LapseAgentMemoryOffersRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		m := agentmemorydb.Memories
		n, err := m.Where(
			m.Status.Eq(agentmemory.MemoryStatusSuggested),
			m.CreatedAt.Lt(req.Before),
		).Update(
			ctx,
			r.db.Keel(ctx),
			m.Status.To(agentmemory.MemoryStatusDismissed),
			m.RetiredAt.To(req.At),
			m.RetiredByUserID.ToNull(),
			m.UpdatedAt.To(req.At),
		)
		if err != nil {
			return 0, fmt.Errorf("lapse agent memory offers: %w", err)
		}

		return n, nil
	})
}
