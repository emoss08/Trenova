package agentwaitrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entityName       = "Wait"
	defaultListLimit = 25
	maxListLimit     = 100
	maxOpenScan      = 500
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

func New(p Params) repositories.AgentWaitRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.agent-wait-repository")}
}

func tenantOf(entity *agentwait.Wait) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func (r *repository) Insert(
	ctx context.Context,
	entity *agentwait.Wait,
) (*agentwait.Wait, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentwait.Wait, error) {
		db := r.db.DBForContext(ctx)
		cols := buncolgen.WaitColumns
		owner := ownerKey(entity)
		if _, err := db.NewRaw(
			"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "agent-wait:"+owner,
		).Exec(ctx); err != nil {
			return nil, fmt.Errorf("hold the agent's waits: %w", err)
		}

		open, err := db.NewSelect().
			Model((*agentwait.Wait)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = buncolgen.WaitScopeTenant(sq, tenantOf(entity)).
					Where(cols.Status.Eq(), agentwait.StatusWaiting)
				if entity.Conversational() {
					return sq.Where(cols.ThreadID.Eq(), entity.ThreadID)
				}
				return sq.
					Where(cols.AgentDefinitionID.Eq(), entity.AgentDefinitionID).
					Where(cols.SubjectID.Eq(), entity.SubjectID)
			}).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count the agent's open waits: %w", err)
		}
		if open >= agentwait.MaxOpenPerOwner {
			return nil, repositories.ErrTooManyWaits
		}

		if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
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
	req *repositories.GetAgentWaitRequest,
) (*agentwait.Wait, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentwait.Wait, error) {
		cols := buncolgen.WaitColumns
		entity := new(agentwait.Wait)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = buncolgen.WaitScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
				if req.ThreadID.IsNotNil() {
					sq = sq.Where(cols.ThreadID.Eq(), req.ThreadID)
				}
				if req.UserID.IsNotNil() {
					sq = sq.Where(cols.UserID.Eq(), req.UserID)
				}
				return sq
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}

		return entity, nil
	})
}

func (r *repository) ListByThread(
	ctx context.Context,
	req *repositories.ListThreadWaitsRequest,
) ([]*agentwait.Wait, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		cols := buncolgen.WaitColumns
		limit := req.Limit
		if limit <= 0 || limit > maxListLimit {
			limit = defaultListLimit
		}
		items := make([]*agentwait.Wait, 0, limit)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.WaitScopeTenant(sq, req.TenantInfo).
					Where(cols.ThreadID.Eq(), req.ThreadID).
					Where(cols.UserID.Eq(), req.UserID)
			}).
			Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(limit).
			Scan(ctx)
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
	req *repositories.ListOpenWaitsWatchingRequest,
) ([]*agentwait.Wait, error) {
	if len(req.Kinds) == 0 || len(req.WatchIDs) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		cols := buncolgen.WaitColumns
		items := make([]*agentwait.Wait, 0, len(req.WatchIDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.WaitScopeTenant(sq, req.TenantInfo).
					Where(cols.Status.Eq(), agentwait.StatusWaiting).
					Where(cols.Kind.In(), bun.List(req.Kinds)).
					Where(cols.WatchID.In(), bun.List(req.WatchIDs))
			}).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("read the waits watching these records: %w", err)
		}

		return items, nil
	})
}

func (r *repository) ListOpenOfKinds(
	ctx context.Context,
	req *repositories.ListOpenWaitsOfKindsRequest,
) ([]*agentwait.Wait, error) {
	if len(req.Kinds) == 0 {
		return nil, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		cols := buncolgen.WaitColumns
		limit := req.Limit
		if limit <= 0 || limit > maxOpenScan {
			limit = maxOpenScan
		}
		items := make([]*agentwait.Wait, 0, min(limit, defaultListLimit))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.WaitScopeTenant(sq, req.TenantInfo).
					Where(cols.Status.Eq(), agentwait.StatusWaiting).
					Where(cols.Kind.In(), bun.List(req.Kinds))
			}).
			Order(cols.CreatedAt.OrderAsc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("read the open waits of these kinds: %w", err)
		}

		return items, nil
	})
}

func (r *repository) ListOpenByThreads(
	ctx context.Context,
	req *repositories.ListOpenWaitsByThreadsRequest,
) (map[pulid.ID][]*agentwait.Wait, error) {
	out := make(map[pulid.ID][]*agentwait.Wait, len(req.ThreadIDs))
	if len(req.ThreadIDs) == 0 {
		return out, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID][]*agentwait.Wait, error) {
		cols := buncolgen.WaitColumns
		items := make([]*agentwait.Wait, 0, len(req.ThreadIDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Column(
				cols.ID.Bare(),
				cols.Kind.Bare(),
				cols.Condition.Bare(),
				cols.ThreadID.Bare(),
				cols.DueAt.Bare(),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.WaitScopeTenant(sq, req.TenantInfo).
					Where(cols.Status.Eq(), agentwait.StatusWaiting).
					Where(cols.UserID.Eq(), req.UserID).
					Where(cols.ThreadID.In(), bun.List(req.ThreadIDs))
			}).
			Order(cols.CreatedAt.OrderAsc()).
			Limit(len(req.ThreadIDs) * agentwait.MaxOpenPerOwner).
			Scan(ctx)
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
	req *repositories.ListOverdueWaitsRequest,
) ([]*agentwait.Wait, error) {
	ctx = dbscope.WithSystem(ctx, "list open agent waits past their expiry in every organization")

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentwait.Wait, error) {
		cols := buncolgen.WaitColumns
		limit := req.Limit
		if limit <= 0 || limit > maxOpenScan {
			limit = maxOpenScan
		}
		items := make([]*agentwait.Wait, 0, min(limit, defaultListLimit))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Where(cols.Status.Eq(), agentwait.StatusWaiting).
			Where(cols.ExpiresAt.Lt(), req.ExpiredBefore).
			Order(cols.ExpiresAt.OrderAsc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("read the overdue waits: %w", err)
		}

		return items, nil
	})
}

func (r *repository) UpdateCondition(
	ctx context.Context,
	req *repositories.UpdateAgentWaitConditionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.WaitColumns
		_, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agentwait.Wait)(nil)).
			Set(cols.Condition.Set(), req.Condition).
			Set(cols.Version.Inc(1)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.WaitScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Status.Eq(), agentwait.StatusWaiting)
			}).
			Exec(ctx)

		return err
	})
}

func (r *repository) SetDue(ctx context.Context, req *repositories.SetAgentWaitDueRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.WaitColumns
		q := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agentwait.Wait)(nil)).
			Set(cols.DueAt.Set(), req.DueAt).
			Set(cols.Version.Inc(1))
		if req.WorkflowID != "" {
			q = q.Set(cols.WorkflowID.Set(), req.WorkflowID)
		}
		_, err := q.
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.WaitScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Exec(ctx)

		return err
	})
}

func (r *repository) Resolve(
	ctx context.Context,
	req *repositories.ResolveAgentWaitRequest,
) (*agentwait.Wait, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*agentwait.Wait, bool, error) {
		cols := buncolgen.WaitColumns
		entity := new(agentwait.Wait)
		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Set(cols.Status.Set(), req.Status).
			Set(cols.Outcome.Set(), req.Outcome).
			Set(cols.ResolvedAt.Set(), req.ResolvedAt).
			Set(cols.DueAt.SetNull()).
			Set(cols.Version.Inc(1)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.WaitScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Status.Eq(), agentwait.StatusWaiting)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, false, err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return nil, false, err
		}
		if changed > 0 {
			return entity, true, nil
		}

		current, err := r.Get(ctx, &repositories.GetAgentWaitRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})

		return current, false, err
	})
}

func (r *repository) MarkResumed(
	ctx context.Context,
	req *repositories.MarkAgentWaitResumedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.WaitColumns
		_, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agentwait.Wait)(nil)).
			Set(cols.ResumedTurnID.Set(), req.TurnID).
			Set(cols.ResumedRunID.Set(), req.RunID).
			Set(cols.Version.Inc(1)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.WaitScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Exec(ctx)

		return err
	})
}
