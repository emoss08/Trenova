package desktranscriptrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	cursorResource   = "desk_transcripts:"
	maxMessageRead   = 10_000
	maxConversations = 1000
)

type Params struct {
	fx.In

	Entitlements entitlementports.Filter
	DB           *postgres.Connection
	Logger       *zap.Logger
}

type conversations struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewConversations(p Params) desktranscriptports.ConversationRepository {
	return &conversations{
		db: p.DB,
		l:  p.Logger.Named("postgres.desk-transcript-conversations"),
	}
}

func summaryColumns() []keel.Expr {
	th := conversationdb.Threads

	return []keel.Expr{
		th.ID,
		th.UserID,
		th.AgentDefinitionID,
		th.Title,
		th.Origin,
		th.Status,
		th.CreatedAt,
		th.LastMessageAt,
	}
}

func matching(
	filter *desktranscriptports.MatchConversationsRequest,
	query string,
) []keel.Pred {
	th := conversationdb.Threads
	f := filter.Filter
	window := filter.Window
	query = strings.TrimSpace(query)

	return []keel.Pred{
		keel.When(f.UserID.IsNotNil(), th.UserID.Eq(f.UserID)),
		keel.When(f.AgentDefinitionID.IsNotNil(), th.AgentDefinitionID.Eq(f.AgentDefinitionID)),
		keel.When(len(f.Origins) > 0, th.Origin.In(f.Origins...)),
		keel.When(window.From > 0, th.LastMessageAt.Gte(window.From)),
		keel.When(window.Before > 0, th.LastMessageAt.Lt(window.Before)),
		keel.When(query != "", th.Title.ContainsFold(query)),
	}
}

func newestFirst() []keel.Order {
	th := conversationdb.Threads

	return []keel.Order{th.LastMessageAt.Desc(), th.ID.Desc()}
}

func cursorBinding(req *desktranscriptports.ListConversationsRequest) pagination.CursorBinding {
	return pagination.BindCursor(
		req.TenantInfo,
		cursorResource+req.Filter.Key()+"|"+strings.TrimSpace(req.Query),
		nil,
	)
}

func (r *conversations) List(
	ctx context.Context,
	req *desktranscriptports.ListConversationsRequest,
) (*desktranscriptports.ConversationPage, error) {
	binding := cursorBinding(req)
	after := ""
	if req.Cursor.After != "" {
		decoded, err := pagination.DecodeKeysetCursor(binding, req.Cursor.After)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"after",
				errortypes.ErrInvalidFormat,
				"Cursor is not one a previous page returned",
			)
		}
		after = decoded
	}

	limit := req.Cursor.Limit
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}
	pageRequest := keel.After(after, limit)
	if req.Cursor.IncludeTotalCount {
		pageRequest = pageRequest.Total(pagination.ExactTotalUpTo).Estimate()
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*desktranscriptports.ConversationPage, error) {
			match := &desktranscriptports.MatchConversationsRequest{
				TenantInfo: req.TenantInfo,
				Filter:     req.Filter,
				Window:     req.Window,
			}
			th := conversationdb.Threads
			page, err := th.Pick(summaryColumns()...).
				Where(matching(match, req.Query)...).
				Order(newestFirst()...).
				Page(ctx, r.db.Keel(ctx), pageRequest)
			if err != nil {
				r.l.Error("failed to list desk conversations", zap.Error(err))
				return nil, fmt.Errorf("list desk conversations: %w", err)
			}

			cursors := make([]string, len(page.Items))
			for idx := range page.Items {
				cursor, encodeErr := pagination.EncodeKeysetCursor(
					binding,
					keel.CursorAt(page, idx),
				)
				if encodeErr != nil {
					return nil, encodeErr
				}
				cursors[idx] = cursor
			}

			result := &desktranscriptports.ConversationPage{
				Items:       page.Items,
				EdgeCursors: cursors,
				HasNextPage: page.Next != "",
			}
			if req.Cursor.IncludeTotalCount {
				total := int(page.Total)
				result.TotalCount = &total
				result.TotalExact = page.TotalExact
			}

			return result, nil
		},
	)
}

func (r *conversations) CountUpTo(
	ctx context.Context,
	req *desktranscriptports.MatchConversationsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		th := conversationdb.Threads
		n, err := th.Where(matching(req, "")...).
			CountUpTo(ctx, r.db.Keel(ctx), int64(req.Limit))
		if err != nil {
			return 0, fmt.Errorf("count desk conversations: %w", err)
		}

		return int(n), nil
	})
}

func (r *conversations) Match(
	ctx context.Context,
	req *desktranscriptports.MatchConversationsRequest,
) ([]desktranscriptports.ConversationRef, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]desktranscriptports.ConversationRef, error) {
			th := conversationdb.Threads
			rows, err := th.Pick(th.ID, th.UserID, th.LastMessageAt).
				Where(matching(req, "")...).
				Order(newestFirst()...).
				Limit(req.Limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("match desk conversations: %w", err)
			}

			refs := make([]desktranscriptports.ConversationRef, len(rows))
			for idx, row := range rows {
				refs[idx] = desktranscriptports.ConversationRef{ID: row.ID, UserID: row.UserID}
			}

			return refs, nil
		},
	)
}

func (r *conversations) Get(
	ctx context.Context,
	req desktranscriptports.GetConversationRequest,
) (*conversation.Thread, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.Thread, error) {
		th := conversationdb.Threads
		thread, err := th.Pick(summaryColumns()...).
			Where(th.ID.Eq(req.ID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Conversation")
		}

		return thread, nil
	})
}

func (r *conversations) ListByIDs(
	ctx context.Context,
	req desktranscriptports.ConversationsByIDsRequest,
) ([]*conversation.Thread, error) {
	if len(req.IDs) == 0 {
		return []*conversation.Thread{}, nil
	}
	if len(req.IDs) > maxConversations {
		return nil, fmt.Errorf(
			"list desk conversations by ids: %d ids is over %d",
			len(req.IDs),
			maxConversations,
		)
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.Thread, error) {
		th := conversationdb.Threads
		threads, err := th.Pick(summaryColumns()...).
			Where(th.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list desk conversations by ids: %w", err)
		}

		return threads, nil
	})
}

type threadMessageCount struct {
	ThreadID pulid.ID
	Total    int64
}

func (r *conversations) CountMessages(
	ctx context.Context,
	req desktranscriptports.ConversationsByIDsRequest,
) (map[pulid.ID]int, error) {
	counts := make(map[pulid.ID]int, len(req.IDs))
	if len(req.IDs) == 0 {
		return counts, nil
	}
	if len(req.IDs) > maxConversations {
		return nil, fmt.Errorf(
			"count desk conversation messages: %d ids is over %d",
			len(req.IDs),
			maxConversations,
		)
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		m := conversationdb.Messages
		rows, err := keel.Into[threadMessageCount](ctx, r.db.Keel(ctx),
			m.Pick(m.ThreadID, keel.Count()).
				Where(m.ThreadID.In(req.IDs...)).
				Group(m.ThreadID),
		)
		if err != nil {
			return nil, fmt.Errorf("count desk conversation messages: %w", err)
		}
		for _, row := range rows {
			counts[row.ThreadID] = int(row.Total)
		}

		return counts, nil
	})
}

func (r *conversations) ListMessages(
	ctx context.Context,
	req *desktranscriptports.ThreadMessagesRequest,
) ([]conversation.Message, error) {
	if len(req.ThreadIDs) == 0 {
		return []conversation.Message{}, nil
	}
	limit := req.Limit
	if limit <= 0 || limit > maxMessageRead {
		limit = maxMessageRead
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]conversation.Message, error) {
		m := conversationdb.Messages
		rows, err := m.Where(m.ThreadID.In(req.ThreadIDs...)).
			Order(m.ThreadID.Asc(), m.Sequence.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list desk conversation messages: %w", err)
		}

		messages := make([]conversation.Message, len(rows))
		for idx, row := range rows {
			messages[idx] = *row
		}

		return messages, nil
	})
}
