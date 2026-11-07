package conversationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
)

// NewConversationActivity is the conversation repository as a wait reads it.
func NewConversationActivity(p Params) agentwaitports.ConversationActivity {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.conversation-activity")}
}

// recordReadQuery finds a tool result in the conversation, after the time
// given, that carries the record's id: the turn read the record as it stood
// then. It reads one conversation's later messages through
// idx_assistant_messages_thread.
func recordReadQuery(
	req *agentwaitports.RecordReadSinceRequest,
) keel.Query[*conversation.Message] {
	m := conversationdb.Messages

	return m.Where(
		m.OrganizationID.Eq(req.TenantInfo.OrgID),
		m.BusinessUnitID.Eq(req.TenantInfo.BuID),
		m.ThreadID.Eq(req.ThreadID),
		m.Role.Eq(conversation.RoleTool),
		m.ToolFailed.IsFalse(),
		m.CreatedAt.Gt(req.Since),
		m.Content.Contains(req.RecordID.String()),
	)
}

func (r *repository) RecordReadSince(
	ctx context.Context,
	req *agentwaitports.RecordReadSinceRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		read, err := recordReadQuery(req).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("read whether the conversation read the record: %w", err)
		}

		return read, nil
	})
}
