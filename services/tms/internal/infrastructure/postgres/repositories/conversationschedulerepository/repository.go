package conversationschedulerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 25
	maxListLimit     = 100
	maxAcrossLimit   = 500
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

func New(p Params) repositories.ConversationScheduleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.conversation-schedule-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	req repositories.CreateConversationScheduleRequest,
) (*conversationschedule.Schedule, *conversation.Message, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (
		*conversationschedule.Schedule, *conversation.Message, error,
	) {
		schedule := req.Schedule
		message := req.Message

		err := r.db.DBForContext(ctx).RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
			if _, err := tx.NewInsert().Model(schedule).Returning("*").Exec(txCtx); err != nil {
				return fmt.Errorf("insert schedule: %w", err)
			}

			cols := buncolgen.MessageColumns
			var maxSequence int
			err := tx.NewSelect().
				Model((*conversation.Message)(nil)).
				ColumnExpr("COALESCE(MAX(?), -1)", bun.Ident(cols.Sequence.Bare())).
				Where(cols.ThreadID.Eq(), schedule.ThreadID).
				Where(cols.OrganizationID.Eq(), schedule.OrganizationID).
				Where(cols.BusinessUnitID.Eq(), schedule.BusinessUnitID).
				Scan(txCtx, &maxSequence)
			if err != nil {
				return fmt.Errorf("read current sequence: %w", err)
			}

			now := timeutils.NowUnix()
			message.ThreadID = schedule.ThreadID
			message.OrganizationID = schedule.OrganizationID
			message.BusinessUnitID = schedule.BusinessUnitID
			message.ScheduleID = schedule.ID
			message.Sequence = maxSequence + 1
			if message.CreatedAt == 0 {
				message.CreatedAt = now
			}
			if _, err = tx.NewInsert().Model(&message).Returning("*").Exec(txCtx); err != nil {
				return fmt.Errorf("insert schedule message: %w", err)
			}

			threadCols := buncolgen.ThreadColumns
			if _, err = tx.NewUpdate().
				Model((*conversation.Thread)(nil)).
				Where(threadCols.ID.Eq(), schedule.ThreadID).
				Where(threadCols.OrganizationID.Eq(), schedule.OrganizationID).
				Where(threadCols.BusinessUnitID.Eq(), schedule.BusinessUnitID).
				Set(threadCols.LastMessageAt.Set(), now).
				Set(threadCols.UpdatedAt.Set(), now).
				Exec(txCtx); err != nil {
				return fmt.Errorf("touch thread: %w", err)
			}

			return nil
		})
		if err != nil {
			r.l.Error("failed to create conversation schedule", zap.Error(err))
			return nil, nil, err
		}

		return schedule, &message, nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetConversationScheduleRequest,
) (*conversationschedule.Schedule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversationschedule.Schedule, error) {
		cols := buncolgen.ScheduleColumns
		entity := new(conversationschedule.Schedule)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = buncolgen.ScheduleScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
				if req.UserID.IsNotNil() {
					sq = sq.Where(cols.UserID.Eq(), req.UserID)
				}

				return sq
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Schedule")
		}

		return entity, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req repositories.ListConversationSchedulesRequest,
) (*pagination.ListResult[*conversationschedule.Schedule], error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (
		*pagination.ListResult[*conversationschedule.Schedule], error,
	) {
		cols := buncolgen.ScheduleColumns

		limit := req.Limit
		if limit <= 0 {
			limit = defaultListLimit
		}
		limit = min(limit, maxListLimit)

		entities := make([]*conversationschedule.Schedule, 0, limit)
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return r.scoped(sq, req.TenantInfo, req.UserID, req.ThreadID)
			}).
			Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(limit).
			Offset(max(req.Offset, 0)).
			ScanAndCount(ctx)
		if err != nil {
			r.l.Error("failed to list conversation schedules", zap.Error(err))
			return nil, err
		}

		return &pagination.ListResult[*conversationschedule.Schedule]{
			Items: entities,
			Total: total,
		}, nil
	})
}

func (r *repository) Count(
	ctx context.Context,
	req repositories.CountConversationSchedulesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		return r.db.DBForContext(ctx).
			NewSelect().
			Model((*conversationschedule.Schedule)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return r.scoped(sq, req.TenantInfo, req.UserID, req.ThreadID)
			}).
			Count(ctx)
	})
}

// scoped is one person's schedules, in one conversation when a thread is
// named. A person is always named: schedules are read as their owner's.
func (r *repository) scoped(
	sq *bun.SelectQuery,
	tenant pagination.TenantInfo,
	userID, threadID pulid.ID,
) *bun.SelectQuery {
	cols := buncolgen.ScheduleColumns
	sq = buncolgen.ScheduleScopeTenant(sq, tenant).Where(cols.UserID.Eq(), userID)
	if threadID.IsNotNil() {
		sq = sq.Where(cols.ThreadID.Eq(), threadID)
	}

	return sq
}

func (r *repository) ListAcrossTenants(
	ctx context.Context,
	req repositories.ListConversationSchedulesAcrossTenantsRequest,
) ([]*conversationschedule.Schedule, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list conversation schedules across every organization to reconcile them",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversationschedule.Schedule, error) {
		cols := buncolgen.ScheduleColumns

		limit := req.Limit
		if limit <= 0 || limit > maxAcrossLimit {
			limit = maxAcrossLimit
		}

		entities := make([]*conversationschedule.Schedule, 0, limit)
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Order(cols.ID.OrderAsc()).
			Limit(limit)
		if req.AfterID.IsNotNil() {
			query = query.Where(cols.ID.Gt(), req.AfterID)
		}
		if err := query.Scan(ctx); err != nil {
			return nil, fmt.Errorf("list conversation schedules: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) UpdateState(
	ctx context.Context,
	schedule *conversationschedule.Schedule,
) (*conversationschedule.Schedule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversationschedule.Schedule, error) {
		cols := buncolgen.ScheduleColumns
		ov := schedule.Version
		schedule.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(schedule).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ScheduleScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: schedule.OrganizationID,
					BuID:  schedule.BusinessUnitID,
				}).
					Where(cols.ID.Eq(), schedule.ID).
					Where(cols.UserID.Eq(), schedule.UserID).
					Where(cols.Version.Eq(), ov)
			}).
			Set(cols.Enabled.Set(), schedule.Enabled).
			Set(cols.NextRunAt.Set(), schedule.NextRunAt).
			Set(cols.Version.Set(), schedule.Version).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to update conversation schedule", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(res, "Schedule", schedule.ID.String()); err != nil {
			return nil, err
		}

		return schedule, nil
	})
}

func (r *repository) RecordRun(
	ctx context.Context,
	req repositories.RecordConversationScheduleRunRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.ScheduleColumns

		_, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*conversationschedule.Schedule)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ScheduleScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Set(cols.LastRunAt.Set(), req.RunAt).
			Set(cols.NextRunAt.Set(), req.NextRunAt).
			Set(cols.LastTurnID.Set(), req.TurnID).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("record schedule run: %w", err)
		}

		return nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.GetConversationScheduleRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req.UserID.IsNil() {
			return errortypes.NewBusinessError("A schedule is deleted by its owner")
		}
		cols := buncolgen.ScheduleColumns

		res, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*conversationschedule.Schedule)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ScheduleScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.UserID.Eq(), req.UserID)
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete schedule: %w", err)
		}

		return dberror.CheckRowsAffected(res, "Schedule", req.ID.String())
	})
}
