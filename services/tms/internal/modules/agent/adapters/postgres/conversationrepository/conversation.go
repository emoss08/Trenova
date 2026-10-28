package conversationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultThreadLimit  = 50
	maxThreadLimit      = 100
	maxThreadRangeLimit = 1000
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

func New(p Params) assistantports.ConversationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.conversation-repository"),
	}
}

func (r *repository) CreateThread(
	ctx context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.Thread, error) {
		if err := conversationdb.Threads.Create(ctx, r.db.Keel(ctx), thread); err != nil {
			r.l.Error("failed to create thread", zap.Error(err))
			return nil, err
		}

		return thread, nil
	})
}

func (r *repository) GetThread(
	ctx context.Context,
	req assistantports.GetThreadRequest,
) (*conversation.Thread, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.Thread, error) {
		th := conversationdb.Threads
		entity, err := th.Where(th.ID.Eq(req.ID), th.UserID.Eq(req.UserID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Thread")
		}

		return entity, nil
	})
}

func (r *repository) GetThreadOwned(
	ctx context.Context,
	req assistantports.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.Thread, error) {
		entity, err := conversationdb.Threads.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Thread")
		}

		return entity, nil
	})
}

func (r *repository) ListThreadAgentsByIDs(
	ctx context.Context,
	req assistantports.ListThreadAgentsByIDsRequest,
) ([]*conversation.Thread, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.Thread, error) {
		if len(req.IDs) == 0 {
			return []*conversation.Thread{}, nil
		}

		th := conversationdb.Threads
		threads, err := th.Pick(th.ID, th.OrganizationID, th.BusinessUnitID, th.AgentDefinitionID).
			Where(th.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list thread agents by ids: %w", err)
		}

		return threads, nil
	})
}

func (r *repository) ListThreads(
	ctx context.Context,
	req assistantports.ListThreadsRequest,
) (*assistantports.ThreadPage, error) {
	window, err := threadWindowOf(&req)
	if err != nil {
		return nil, err
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*assistantports.ThreadPage, error) {
		entities, listErr := threadPageQuery(&req, window).List(ctx, r.db.Keel(ctx))
		if listErr != nil {
			r.l.Error("failed to list threads", zap.Error(listErr))
			return nil, listErr
		}

		page := &assistantports.ThreadPage{Items: entities}
		switch {
		case len(entities) > window.limit:
			page.Items = entities[:window.limit]
			page.NextCursor = threadCursorAt(page.Items[window.limit-1]).encode()
		case window.until != nil:
			page.NextCursor = req.Until
		}

		return page, nil
	})
}

type threadWindow struct {
	after *threadCursor
	until *threadCursor
	limit int
}

func threadWindowOf(req *assistantports.ListThreadsRequest) (threadWindow, error) {
	window := threadWindow{limit: req.Limit}
	if window.limit <= 0 {
		window.limit = defaultThreadLimit
	}
	window.limit = min(window.limit, maxThreadLimit)

	if req.Cursor != "" {
		after, err := decodeThreadCursor(req.Cursor)
		if err != nil {
			return threadWindow{}, err
		}
		window.after = after
	}
	if req.Until != "" {
		until, err := decodeThreadCursor(req.Until)
		if err != nil {
			return threadWindow{}, err
		}
		window.until = until
		window.limit = maxThreadRangeLimit
	}

	return window, nil
}

func threadTouchedAt() keel.Int64Expr {
	th := conversationdb.Threads

	return th.LastMessageAt.Greatest(th.CreatedAt)
}

func threadListScope(userID pulid.ID, includeUnlisted bool) keel.Pred {
	th := conversationdb.Threads

	return keel.And(
		th.UserID.Eq(userID),
		keel.When(!includeUnlisted, th.Origin.NotIn(conversation.UnlistedOrigins()...)),
	)
}

func threadPositionPred(
	touchedAt keel.Expr,
	compare func(keel.TupleExpr, keel.TupleExpr) keel.Pred,
	cursor *threadCursor,
) keel.Pred {
	if cursor == nil {
		return nil
	}

	th := conversationdb.Threads

	return compare(
		keel.Tuple(th.Pinned, touchedAt, th.ID),
		keel.Tuple(cursor.pinned, cursor.touchedAt, cursor.id.String()),
	)
}

func threadPageQuery(
	req *assistantports.ListThreadsRequest,
	window threadWindow,
) *keel.Query[*conversation.Thread] {
	th := conversationdb.Threads
	touchedAt := threadTouchedAt()

	var pinned keel.Pred
	if req.Pinned != nil {
		pinned = th.Pinned.Eq(*req.Pinned)
	}

	return th.Where(
		threadListScope(req.UserID, req.IncludeUnlisted),
		pinned,
		keel.When(req.TouchedFrom > 0, touchedAt.Gte(req.TouchedFrom)),
		keel.When(req.TouchedBefore > 0, touchedAt.Lt(req.TouchedBefore)),
		threadPositionPred(touchedAt, keel.TupleExpr.Lt, window.after),
		threadPositionPred(touchedAt, keel.TupleExpr.Gte, window.until),
	).
		Order(th.Pinned.Desc(), keel.Desc(touchedAt), th.ID.Desc()).
		Limit(window.limit + 1)
}

type threadShelfBand struct {
	Band  int32
	Total int64
}

func threadShelfBandExpr(edges []int64) keel.Expr {
	th := conversationdb.Threads
	touchedAt := threadTouchedAt()
	band := keel.CaseInt64().When(keel.BoolOf(th.Pinned), keel.Int64(-1))
	for index, edge := range edges {
		band = band.When(touchedAt.Gte(edge), keel.Int64(int64(index)))
	}

	return band.Else(keel.Int64(int64(len(edges))))
}

func threadShelfCountQuery(
	req *assistantports.CountThreadShelvesRequest,
) *keel.Query[*conversation.Thread] {
	th := conversationdb.Threads
	band := threadShelfBandExpr(req.Edges)

	return th.Pick(band, keel.Count()).
		Where(threadListScope(req.UserID, false)).
		Group(band)
}

func (r *repository) CountThreadShelves(
	ctx context.Context,
	req *assistantports.CountThreadShelvesRequest,
) (*assistantports.ThreadShelfCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*assistantports.ThreadShelfCounts, error) {
			rows, err := keel.Into[threadShelfBand](ctx, r.db.Keel(ctx), threadShelfCountQuery(req))
			if err != nil {
				r.l.Error("failed to count thread shelves", zap.Error(err))
				return nil, err
			}

			counts := &assistantports.ThreadShelfCounts{Bands: make([]int, len(req.Edges)+1)}
			for _, row := range rows {
				switch {
				case row.Band < 0:
					counts.Pinned = int(row.Total)
				case int(row.Band) < len(counts.Bands):
					counts.Bands[row.Band] = int(row.Total)
				}
			}

			return counts, nil
		},
	)
}

func keepThreadVersion() keel.Assignment {
	th := conversationdb.Threads

	return th.Version.ToExpr(th.Version)
}

func keepThreadUpdatedAt() keel.Assignment {
	th := conversationdb.Threads

	return th.UpdatedAt.ToExpr(th.UpdatedAt)
}

func (r *repository) UpdateThread(
	ctx context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.Thread, error) {
		th := conversationdb.Threads
		n, err := th.Where(
			th.ID.Eq(thread.ID),
			th.UserID.Eq(thread.UserID),
			th.Version.Eq(thread.Version),
		).Update(ctx, r.db.Keel(ctx),
			th.Title.To(thread.Title),
			th.Status.To(thread.Status),
			th.PreferredProviderID.To(thread.PreferredProviderID),
			th.Origin.To(thread.Origin),
			th.Pinned.To(thread.Pinned),
			th.PinnedFacts.To(thread.PinnedFacts),
			th.AutoCompactOff.To(thread.AutoCompactOff),
			th.SubjectType.To(thread.SubjectType),
			th.SubjectID.To(thread.SubjectID),
			listquery.OptionalInt64(th.SnoozedUntil, thread.SnoozedUntil),
			th.SnoozeAnchor.To(thread.SnoozeAnchor),
			th.SnoozeStopID.To(thread.SnoozeStopID),
		)
		if err != nil {
			return nil, fmt.Errorf("update thread: %w", err)
		}

		if err = dberror.CheckAffected(n, "Thread", thread.ID.String()); err != nil {
			return nil, err
		}

		thread.Version++

		return thread, nil
	})
}

func (r *repository) MarkThreadTainted(
	ctx context.Context,
	req assistantports.MarkThreadTaintedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if !req.Taint.Tainted() {
			return nil
		}

		th := conversationdb.Threads
		n, err := th.Where(th.ID.Eq(req.ThreadID)).Update(ctx, r.db.Keel(ctx),
			th.Taint.To(req.Taint),
			th.TaintedAt.ToExpr(th.TaintedAt.Coalesce(req.TaintedAt)),
			keepThreadVersion(),
			keepThreadUpdatedAt(),
		)
		if err != nil {
			return fmt.Errorf("mark thread tainted: %w", err)
		}

		return dberror.CheckAffected(n, "Thread", req.ThreadID.String())
	})
}

func (r *repository) UpdateThreadContext(
	ctx context.Context,
	req assistantports.UpdateThreadContextRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req.Usage == nil && req.AutoCompactOff == nil && req.WorkingSet == nil {
			return nil
		}

		th := conversationdb.Threads
		sets := []keel.Assignment{keepThreadVersion(), keepThreadUpdatedAt()}
		if req.Usage != nil {
			sets = append(sets, th.ContextUsage.To(req.Usage))
		}
		if req.AutoCompactOff != nil {
			sets = append(sets, th.AutoCompactOff.To(*req.AutoCompactOff))
		}
		if req.WorkingSet != nil {
			sets = append(sets, th.WorkingSet.To(*req.WorkingSet))
		}

		n, err := th.Where(th.ID.Eq(req.ThreadID)).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return fmt.Errorf("update thread context: %w", err)
		}

		return dberror.CheckAffected(n, "Thread", req.ThreadID.String())
	})
}

func (r *repository) AddSavedMemories(
	ctx context.Context,
	req assistantports.AddSavedMemoriesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.Memories) == 0 {
			return nil
		}

		m := conversationdb.Messages
		message, err := m.Pick(m.ID, m.SavedMemories).
			Where(m.ID.Eq(req.MessageID), m.ThreadID.Eq(req.ThreadID)).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return dberror.HandleNotFoundError(err, "Message")
		}

		saved := conversation.MergeSavedMemories(message.SavedMemories, req.Memories)
		if len(saved) == len(message.SavedMemories) {
			return nil
		}

		n, err := m.Where(m.ID.Eq(req.MessageID)).
			Update(ctx, r.db.Keel(ctx), m.SavedMemories.To(saved))
		if err != nil {
			return fmt.Errorf("add saved memories to a message: %w", err)
		}

		return dberror.CheckAffected(n, "Message", req.MessageID.String())
	})
}

func (r *repository) DeleteThread(
	ctx context.Context,
	req assistantports.GetThreadRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		m := conversationdb.Messages
		if _, err := m.Where(m.ThreadID.Eq(req.ID)).Delete(ctx, db); err != nil {
			return err
		}

		th := conversationdb.Threads
		n, err := th.Where(th.ID.Eq(req.ID), th.UserID.Eq(req.UserID)).Delete(ctx, db)
		if err != nil {
			return err
		}

		return dberror.CheckAffected(n, "Thread", req.ID.String())
	})
}

func messagesQuery(req *assistantports.ListMessagesRequest) *keel.Query[*conversation.Message] {
	m := conversationdb.Messages

	var before, after, sinceCompaction keel.Pred
	if req.BeforeSequence != nil {
		before = m.Sequence.Lt(int64(*req.BeforeSequence))
	}
	if req.AfterSequence != nil {
		after = m.Sequence.Gt(int64(*req.AfterSequence))
	}
	if req.SinceCompaction {
		sinceCompaction = afterLatestCompaction(req.ThreadID)
	}

	return m.Where(
		m.ThreadID.Eq(req.ThreadID),
		before,
		after,
		keel.When(req.TurnID.IsNotNil(), m.TurnID.Eq(req.TurnID)),
		keel.When(len(req.ExcludeKinds) > 0, m.Kind.NotIn(req.ExcludeKinds...)),
		keel.When(len(req.Kinds) > 0, m.Kind.In(req.Kinds...)),
		sinceCompaction,
	)
}

func afterLatestCompaction(threadID pulid.ID) keel.Pred {
	m := conversationdb.Messages
	c := conversationdb.Messages.As("compacted")
	through := c.Pick(keel.CastInt64(c.Compaction.Text("through"))).
		Where(c.ThreadID.Eq(threadID), c.Kind.Eq(conversation.MessageKindCompaction)).
		Order(c.Sequence.Desc()).
		Limit(1)

	return keel.Int64Of(m.Sequence).GtExpr(keel.Int64Of(keel.Scalar(through)).Coalesce(-1))
}

func (r *repository) ListMessages(
	ctx context.Context,
	req assistantports.ListMessagesRequest,
) ([]conversation.Message, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]conversation.Message, error) {
		m := conversationdb.Messages
		query := messagesQuery(&req)

		if req.Limit > 0 {
			rows, err := query.Order(m.Sequence.Desc()).Limit(req.Limit).List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}
			messages := derefMessages(rows)
			reverse(messages)

			return messages, nil
		}

		rows, err := query.Order(m.Sequence.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return derefMessages(rows), nil
	})
}

func derefMessages(rows []*conversation.Message) []conversation.Message {
	messages := make([]conversation.Message, len(rows))
	for idx, row := range rows {
		messages[idx] = *row
	}

	return messages
}

func (r *repository) CountMessages(
	ctx context.Context,
	req assistantports.CountMessagesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		m := conversationdb.Messages
		count, err := m.Where(m.ThreadID.Eq(req.ThreadID)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count messages: %w", err)
		}

		return int(count), nil
	})
}

func (r *repository) CarriesAttachment(
	ctx context.Context,
	req *assistantports.CarriesAttachmentRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		return carriesAttachmentQuery(req).Exists(ctx, r.db.Keel(ctx))
	})
}

func carriesAttachmentQuery(
	req *assistantports.CarriesAttachmentRequest,
) *keel.Query[*conversation.Message] {
	m := conversationdb.Messages

	return m.Where(
		m.ThreadID.Eq(req.ThreadID),
		m.Role.Eq(conversation.RoleUser),
		m.Attachments.Contains([]map[string]string{{"documentId": req.DocumentID.String()}}),
	)
}

func reverse(messages []conversation.Message) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}

func nextSequence(ctx context.Context, db *keel.DB, threadID pulid.ID) (int, error) {
	m := conversationdb.Messages
	current, err := keel.IntoOne[*int64](
		ctx,
		db,
		m.Pick(m.Sequence.Max()).Where(m.ThreadID.Eq(threadID)),
	)
	if err != nil {
		return 0, fmt.Errorf("read current sequence: %w", err)
	}
	if current == nil {
		return 0, nil
	}

	return int(*current) + 1, nil
}

func (r *repository) AppendTurn(
	ctx context.Context,
	req assistantports.AppendTurnRequest,
) ([]conversation.Message, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]conversation.Message, error) {
		if len(req.Messages) == 0 {
			return nil, nil
		}

		db := r.db.Keel(ctx)
		first, err := nextSequence(ctx, db, req.ThreadID)
		if err != nil {
			return nil, err
		}

		now := timeutils.NowUnix()
		batch := make([]conversation.Message, 0, len(req.Messages))
		for idx := range req.Messages {
			msg := req.Messages[idx]
			msg.ThreadID = req.ThreadID
			msg.OrganizationID = req.TenantInfo.OrgID
			msg.BusinessUnitID = req.TenantInfo.BuID
			msg.Sequence = first + idx
			if msg.TurnID.IsNil() {
				msg.TurnID = req.TurnID
			}
			batch = append(batch, msg)
		}
		conversation.StampUnstamped(batch, now)

		if err = conversationdb.Messages.CreateMany(
			ctx,
			db,
			sliceutils.Pointers(batch),
		); err != nil {
			return nil, fmt.Errorf("insert turn: %w", err)
		}

		th := conversationdb.Threads
		if _, err = th.Where(th.ID.Eq(req.ThreadID)).
			Update(ctx, db, th.LastMessageAt.To(now), keepThreadVersion()); err != nil {
			return nil, fmt.Errorf("touch thread: %w", err)
		}

		return batch, nil
	})
}

func staleThreadsQuery(
	req *assistantports.DeleteStaleThreadsRequest,
	limit int,
) *keel.Query[*conversation.Thread] {
	th := conversationdb.Threads

	return th.AllTenants().
		Pick(th.ID, th.OrganizationID, th.BusinessUnitID).
		Where(
			th.Origin.Eq(req.Origin),
			th.LastMessageAt.Lt(req.Before),
			th.CreatedAt.Lt(req.Before),
			keel.When(req.SubjectlessOnly, th.SubjectID.IsNull()),
		).
		Order(th.LastMessageAt.Asc()).
		Limit(limit)
}

func (r *repository) DeleteStaleThreads(
	ctx context.Context,
	req assistantports.DeleteStaleThreadsRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "delete stale ask threads across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if req.Origin == "" || req.Before <= 0 {
			return 0, nil
		}
		limit := req.Limit
		if limit <= 0 || limit > defaultThreadLimit*10 {
			limit = defaultThreadLimit * 10
		}

		deleted, err := r.deleteStaleThreads(ctx, &req, limit)
		if err != nil {
			r.l.Error("failed to delete stale threads",
				zap.String("origin", string(req.Origin)),
				zap.Error(err),
			)

			return 0, err
		}

		return deleted, nil
	})
}

func (r *repository) deleteStaleThreads(
	ctx context.Context,
	req *assistantports.DeleteStaleThreadsRequest,
	limit int,
) (int, error) {
	db := r.db.Keel(ctx)
	stale, err := staleThreadsQuery(req, limit).List(ctx, db)
	if err != nil {
		return 0, err
	}
	if len(stale) == 0 {
		return 0, nil
	}

	ids := make([]pulid.ID, len(stale))
	orgs := make([]pulid.ID, len(stale))
	bus := make([]pulid.ID, len(stale))
	for idx, thread := range stale {
		ids[idx] = thread.ID
		orgs[idx] = thread.OrganizationID
		bus[idx] = thread.BusinessUnitID
	}
	doomed := keel.Unnest(
		"doomed",
		[]string{"id", "organization_id", "business_unit_id"},
		ids,
		orgs,
		bus,
	)

	m := conversationdb.Messages
	if _, err = m.AllTenants().
		With(doomed).
		Join(doomed, keel.And(
			doomed.Col("id").EqCol(m.ThreadID),
			doomed.Col("organization_id").EqCol(m.OrganizationID),
			doomed.Col("business_unit_id").EqCol(m.BusinessUnitID),
		)).
		Where(keel.True()).
		Delete(ctx, db); err != nil {
		return 0, err
	}

	th := conversationdb.Threads
	n, err := th.AllTenants().
		With(doomed).
		Join(doomed, keel.And(
			doomed.Col("id").EqCol(th.ID),
			doomed.Col("organization_id").EqCol(th.OrganizationID),
			doomed.Col("business_unit_id").EqCol(th.BusinessUnitID),
		)).
		Where(keel.True()).
		Delete(ctx, db)
	if err != nil {
		return 0, err
	}

	return int(n), nil
}
