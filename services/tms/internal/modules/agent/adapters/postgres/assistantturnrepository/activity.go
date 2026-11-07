package assistantturnrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn/conversationturndb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
)

// NewConversationActivity is the turn repository as a wait reads it.
func NewConversationActivity(p Params) agentwaitports.ConversationActivity {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.assistantturn-activity")}
}

func personSpokeQuery(
	req *agentwaitports.PersonSpokeSinceRequest,
) keel.Query[*conversationturn.AssistantTurn] {
	at := conversationturndb.AssistantTurns

	return at.Where(
		at.OrganizationID.Eq(req.TenantInfo.OrgID),
		at.BusinessUnitID.Eq(req.TenantInfo.BuID),
		at.ThreadID.Eq(req.ThreadID),
		at.CreatedAt.Gt(req.Since),
		at.Origin.Eq(conversationturn.AssistantTurnOriginPerson),
	)
}

// PersonSpokeSince reports whether the conversation's owner started a turn
// after the time given, read newest first by idx_assistant_turns_thread.
func (r *repository) PersonSpokeSince(
	ctx context.Context,
	req *agentwaitports.PersonSpokeSinceRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		spoke, err := personSpokeQuery(req).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("read whether the person spoke in the conversation: %w", err)
		}

		return spoke, nil
	})
}
