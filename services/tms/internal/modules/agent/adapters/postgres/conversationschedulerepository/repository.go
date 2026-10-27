package conversationschedulerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule/conversationscheduledb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) conversationscheduleports.ConversationScheduleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.conversation-schedule-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	req conversationscheduleports.CreateConversationScheduleRequest,
) (*conversationschedule.Schedule, *conversation.Message, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (
		*conversationschedule.Schedule, *conversation.Message, error,
	) {
		schedule := req.Schedule
		message := req.Message

		if err := r.create(ctx, schedule, &message, req.ThreadTitle); err != nil {
			r.l.Error("failed to create conversation schedule", zap.Error(err))
			return nil, nil, err
		}

		return schedule, &message, nil
	})
}

func (r *repository) create(
	ctx context.Context,
	schedule *conversationschedule.Schedule,
	message *conversation.Message,
	threadTitle string,
) error {
	db := r.db.Keel(ctx)
	if schedule.Timezone == "" {
		schedule.Timezone = conversationschedule.DefaultTimezone
	}
	if err := conversationscheduledb.Schedules.Create(ctx, db, schedule); err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}

	m := conversationdb.Messages
	current, err := keel.IntoOne[*int64](
		ctx,
		db,
		m.Pick(m.Sequence.Max()).Where(m.ThreadID.Eq(schedule.ThreadID)),
	)
	if err != nil {
		return fmt.Errorf("read current sequence: %w", err)
	}
	sequence := 0
	if current != nil {
		sequence = int(*current) + 1
	}

	now := timeutils.NowUnix()
	message.ThreadID = schedule.ThreadID
	message.OrganizationID = schedule.OrganizationID
	message.BusinessUnitID = schedule.BusinessUnitID
	message.ScheduleID = schedule.ID
	message.Sequence = sequence
	if message.CreatedAt == 0 {
		message.CreatedAt = now
	}
	if err = m.Create(ctx, db, message); err != nil {
		return fmt.Errorf("insert schedule message: %w", err)
	}

	th := conversationdb.Threads
	if _, err = th.Where(th.ID.Eq(schedule.ThreadID)).Update(ctx, db,
		th.LastMessageAt.To(now),
		th.Title.ToExpr(keel.CaseString().
			When(
				th.Title.Trim().NullIf("").IsNull(),
				keel.Text(threadTitle).NullIf(""),
			).
			Else(th.Title)),
		th.Version.ToExpr(th.Version),
	); err != nil {
		return fmt.Errorf("touch thread: %w", err)
	}

	return nil
}

func (r *repository) Get(
	ctx context.Context,
	req conversationscheduleports.GetConversationScheduleRequest,
) (*conversationschedule.Schedule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversationschedule.Schedule, error) {
		s := conversationscheduledb.Schedules
		entity, err := s.Where(
			s.ID.Eq(req.ID),
			keel.When(req.UserID.IsNotNil(), s.UserID.Eq(req.UserID)),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Schedule")
		}

		return entity, nil
	})
}

type threadTitle struct {
	ThreadID pulid.ID
	Title    string
}

func (r *repository) List(
	ctx context.Context,
	req conversationscheduleports.ListConversationSchedulesRequest,
) (*pagination.ListResult[*conversationscheduleports.ListedConversationSchedule], error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (
		*pagination.ListResult[*conversationscheduleports.ListedConversationSchedule], error,
	) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultListLimit
		}
		limit = min(limit, maxListLimit)

		page := listPageQuery(&req, limit)
		var (
			rows   []*conversationschedule.Schedule
			total  keel.RowCount
			titles []threadTitle
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.ListInto(page, &rows),
			keel.CountEstimateInto(
				scoped(req.UserID, req.ThreadID),
				pagination.ExactTotalUpTo,
				&total,
			),
			keel.IntoSlice(threadTitlesQuery(page), &titles),
		)
		if err != nil {
			r.l.Error("failed to list conversation schedules", zap.Error(err))
			return nil, err
		}

		titleOf := make(map[pulid.ID]string, len(titles))
		for _, title := range titles {
			titleOf[title.ThreadID] = title.Title
		}

		items := make([]*conversationscheduleports.ListedConversationSchedule, 0, len(rows))
		for _, row := range rows {
			items = append(items, &conversationscheduleports.ListedConversationSchedule{
				Schedule:    *row,
				ThreadTitle: titleOf[row.ThreadID],
			})
		}

		return listquery.Result(items, total), nil
	})
}

func listPageQuery(
	req *conversationscheduleports.ListConversationSchedulesRequest,
	limit int,
) *keel.Query[*conversationschedule.Schedule] {
	s := conversationscheduledb.Schedules
	order := []keel.Order{s.CreatedAt.Desc(), s.ID.Desc()}
	if req.Order == conversationscheduleports.ConversationScheduleOrderNextRun {
		order = append([]keel.Order{s.Enabled.Desc(), s.NextRunAt.Asc().NullsLast()}, order...)
	}

	return scoped(req.UserID, req.ThreadID).
		Order(order...).
		Limit(limit).
		Offset(max(req.Offset, 0))
}

func threadTitlesQuery(
	page *keel.Query[*conversationschedule.Schedule],
) *keel.Query[*conversation.Thread] {
	s := conversationscheduledb.Schedules
	th := conversationdb.Threads

	return th.Pick(th.ID, th.Title.Coalesce("")).
		Where(th.ID.InQuery(page.Clone().Pick(s.ThreadID)))
}

func (r *repository) Count(
	ctx context.Context,
	req conversationscheduleports.CountConversationSchedulesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		n, err := scoped(req.UserID, req.ThreadID).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, err
		}

		return int(n), nil
	})
}

func scoped(userID, threadID pulid.ID) *keel.Query[*conversationschedule.Schedule] {
	s := conversationscheduledb.Schedules

	return s.Where(
		s.UserID.Eq(userID),
		keel.When(threadID.IsNotNil(), s.ThreadID.Eq(threadID)),
	)
}

func (r *repository) ListAcrossTenants(
	ctx context.Context,
	req conversationscheduleports.ListConversationSchedulesAcrossTenantsRequest,
) ([]*conversationschedule.Schedule, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list conversation schedules across every organization to reconile them",
	)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*conversationschedule.Schedule, error) {
			limit := req.Limit
			if limit <= 0 || limit > maxAcrossLimit {
				limit = maxAcrossLimit
			}

			s := conversationscheduledb.Schedules
			entities, err := s.AllTenants().
				Where(keel.When(req.AfterID.IsNotNil(), s.ID.Gt(req.AfterID))).
				Order(s.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list conversation schedules: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) UpdateState(
	ctx context.Context,
	schedule *conversationschedule.Schedule,
) (*conversationschedule.Schedule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversationschedule.Schedule, error) {
		s := conversationscheduledb.Schedules
		rows, err := s.Where(
			s.ID.Eq(schedule.ID),
			s.UserID.Eq(schedule.UserID),
			s.Version.Eq(schedule.Version),
		).UpdateReturning(ctx, r.db.Keel(ctx),
			s.Prompt.To(schedule.Prompt),
			s.Cadence.To(schedule.Cadence),
			s.CronExpression.To(schedule.CronExpression),
			s.Enabled.To(schedule.Enabled),
			listquery.OptionalInt64(s.NextRunAt, schedule.NextRunAt),
		)
		if err != nil {
			r.l.Error("failed to update conversation schedule", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckAffected(
			int64(len(rows)),
			"Schedule",
			schedule.ID.String(),
		); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) RecordRun(
	ctx context.Context,
	req conversationscheduleports.RecordConversationScheduleRunRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := conversationscheduledb.Schedules
		_, err := s.Where(s.ID.Eq(req.ID)).Update(ctx, r.db.Keel(ctx),
			s.LastRunAt.To(req.RunAt),
			listquery.OptionalInt64(s.NextRunAt, req.NextRunAt),
			s.LastTurnID.To(req.TurnID),
			s.Version.ToExpr(s.Version),
		)
		if err != nil {
			return fmt.Errorf("record schedule run: %w", err)
		}

		return nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req conversationscheduleports.GetConversationScheduleRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req.UserID.IsNil() {
			return errortypes.NewBusinessError("A schedule is deleted by its owner")
		}

		s := conversationscheduledb.Schedules
		n, err := s.Where(s.ID.Eq(req.ID), s.UserID.Eq(req.UserID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("delete schedule: %w", err)
		}

		return dberror.CheckAffected(n, "Schedule", req.ID.String())
	})
}
