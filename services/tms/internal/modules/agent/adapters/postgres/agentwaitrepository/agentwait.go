package agentwaitrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait/agentwaitdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entityName       = "Wait"
	defaultListLimit = 25
	maxListLimit     = 100
	maxOpenScan      = 500
)

var waitErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentwaitports.AgentWaitRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.agent-wait-repository")}
}

func (r *repository) Insert(
	ctx context.Context,
	entity *agentwait.Wait,
) (*agentwait.Wait, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentwait.Wait, error) {
		db := r.db.Keel(ctx)
		if err := keel.LockKey(ctx, db, "agent-wait:"+ownerKey(entity)); err != nil {
			return nil, fmt.Errorf("hold the agent's waits: %w", err)
		}

		w := agentwaitdb.Waits
		owner := keel.And(
			w.AgentDefinitionID.Eq(entity.AgentDefinitionID),
			w.SubjectID.Eq(entity.SubjectID),
		)
		if entity.Conversational() {
			owner = w.ThreadID.Eq(entity.ThreadID)
		}

		open, err := w.Where(w.Status.Eq(agentwait.StatusWaiting), owner).Count(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("count the agent's open waits: %w", err)
		}
		if open >= agentwait.MaxOpenPerOwner {
			return nil, agentwaitports.ErrTooManyWaits
		}

		if err = w.Create(ctx, db, entity); err != nil {
			r.l.Error(
				"failed to record a wait",
				zap.String("kind", string(entity.Kind)),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func ownerKey(entity *agentwait.Wait) string {
	if entity.Conversational() {
		return entity.ThreadID.String()
	}

	return entity.AgentDefinitionID.String() + ":" + entity.SubjectID.String()
}

func (r *repository) Get(
	ctx context.Context,
	req *agentwaitports.GetAgentWaitRequest,
) (*agentwait.Wait, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentwait.Wait, error) {
		w := agentwaitdb.Waits
		entity, err := w.Where(
			w.ID.Eq(req.ID),
			keel.When(req.ThreadID.IsNotNil(), w.ThreadID.Eq(req.ThreadID)),
			keel.When(req.UserID.IsNotNil(), w.UserID.Eq(req.UserID)),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, waitErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByThread(
	ctx context.Context,
	req *agentwaitports.ListThreadWaitsRequest,
) ([]*agentwait.Wait, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxListLimit {
			limit = defaultListLimit
		}

		w := agentwaitdb.Waits
		items, err := w.Where(w.ThreadID.Eq(req.ThreadID), w.UserID.Eq(req.UserID)).
			Order(w.CreatedAt.Desc(), w.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list a conversation's waits",
				zap.String("thread", req.ThreadID.String()), zap.Error(err))
			return nil, err
		}

		return items, nil
	})
}

func (r *repository) ListOpenWatching(
	ctx context.Context,
	req *agentwaitports.ListOpenWaitsWatchingRequest,
) ([]*agentwait.Wait, error) {
	if len(req.Kinds) == 0 || len(req.WatchIDs) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		w := agentwaitdb.Waits
		items, err := w.Where(
			w.Status.Eq(agentwait.StatusWaiting),
			w.Kind.In(req.Kinds...),
			w.WatchID.In(req.WatchIDs...),
		).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the waits watching these records: %w", err)
		}

		return items, nil
	})
}

func (r *repository) ListOpenOfKinds(
	ctx context.Context,
	req *agentwaitports.ListOpenWaitsOfKindsRequest,
) ([]*agentwait.Wait, error) {
	if len(req.Kinds) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxOpenScan {
			limit = maxOpenScan
		}

		w := agentwaitdb.Waits
		items, err := w.Where(w.Status.Eq(agentwait.StatusWaiting), w.Kind.In(req.Kinds...)).
			Order(w.CreatedAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the open waits of these kinds: %w", err)
		}

		return items, nil
	})
}

func (r *repository) ListOpenByThreads(
	ctx context.Context,
	req *agentwaitports.ListOpenWaitsByThreadsRequest,
) (map[pulid.ID][]*agentwait.Wait, error) {
	out := make(map[pulid.ID][]*agentwait.Wait, len(req.ThreadIDs))
	if len(req.ThreadIDs) == 0 {
		return out, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID][]*agentwait.Wait, error) {
		w := agentwaitdb.Waits
		items, err := w.Pick(w.ID, w.Kind, w.Condition, w.ThreadID, w.DueAt).
			Where(
				w.Status.Eq(agentwait.StatusWaiting),
				w.UserID.Eq(req.UserID),
				w.ThreadID.In(req.ThreadIDs...),
			).
			Order(w.CreatedAt.Asc()).
			Limit(len(req.ThreadIDs) * agentwait.MaxOpenPerOwner).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the conversations' open waits: %w", err)
		}

		for _, item := range items {
			out[item.ThreadID] = append(out[item.ThreadID], item)
		}

		return out, nil
	})
}

func (r *repository) ListOverdueAcrossTenants(
	ctx context.Context,
	req *agentwaitports.ListOverdueWaitsRequest,
) ([]*agentwait.Wait, error) {
	ctx = dbscope.WithSystem(ctx, "list open agent waits past their expiry in every organization")

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxOpenScan {
			limit = maxOpenScan
		}

		w := agentwaitdb.Waits
		items, err := w.AllTenants().
			Where(w.Status.Eq(agentwait.StatusWaiting), w.ExpiresAt.Lt(req.ExpiredBefore)).
			Order(w.ExpiresAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the overdue waits: %w", err)
		}

		return items, nil
	})
}

func (r *repository) UpdateCondition(
	ctx context.Context,
	req *agentwaitports.UpdateAgentWaitConditionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		w := agentwaitdb.Waits
		_, err := w.Where(w.ID.Eq(req.ID), w.Status.Eq(agentwait.StatusWaiting)).
			Update(ctx, r.db.Keel(ctx), w.Condition.To(req.Condition))

		return err
	})
}

func (r *repository) SetDue(ctx context.Context, req *agentwaitports.SetAgentWaitDueRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		w := agentwaitdb.Waits
		sets := []keel.Assignment{w.DueAt.ToNull()}
		if req.DueAt != nil {
			sets[0] = w.DueAt.To(*req.DueAt)
		}
		if req.WorkflowID != "" {
			sets = append(sets, w.WorkflowID.To(req.WorkflowID))
		}

		_, err := w.Where(w.ID.Eq(req.ID)).Update(ctx, r.db.Keel(ctx), sets...)

		return err
	})
}

func (r *repository) Resolve(
	ctx context.Context,
	req *agentwaitports.ResolveAgentWaitRequest,
) (*agentwait.Wait, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*agentwait.Wait, bool, error) {
		w := agentwaitdb.Waits
		rows, err := w.Where(w.ID.Eq(req.ID), w.Status.Eq(agentwait.StatusWaiting)).
			UpdateReturning(
				ctx,
				r.db.Keel(ctx),
				w.Status.To(req.Status),
				w.Outcome.To(req.Outcome),
				w.ResolvedAt.To(req.ResolvedAt),
				w.DueAt.ToNull(),
			)
		if err != nil {
			return nil, false, err
		}
		if len(rows) > 0 {
			return rows[0], true, nil
		}

		current, err := r.Get(ctx, &agentwaitports.GetAgentWaitRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})

		return current, false, err
	})
}

func (r *repository) MarkResumed(
	ctx context.Context,
	req *agentwaitports.MarkAgentWaitResumedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		w := agentwaitdb.Waits
		_, err := w.Where(w.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			w.ResumedTurnID.To(req.TurnID),
			w.ResumedRunID.To(req.RunID),
		)

		return err
	})
}
