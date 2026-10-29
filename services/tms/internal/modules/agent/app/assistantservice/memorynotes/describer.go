package memorynotes

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

type Describer struct {
	Logger   *zap.Logger
	Memories agentmemoryports.AgentMemoryService
}

// Describe fills each reply's memories as its reader may see them,
// in one read for the whole page. A memory since moved out of the reader's
// reach is left out, and a reply whose memories cannot be read is served
// without them rather than not at all.
func (s *Describer) Describe(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
	messages []conversation.Message,
) {
	if s.Memories == nil || userID.IsNil() {
		return
	}

	ids := make([]pulid.ID, 0)
	for idx := range messages {
		ids = append(ids, messageMemoryIDs(&messages[idx])...)
	}
	if len(ids) == 0 {
		return
	}

	actor := s.deskMemoryActor(ctx, tenant, userID)
	found, err := s.Memories.DeskByIDs(ctx, actor, sliceutils.Dedupe(ids))
	if err != nil {
		s.Logger.Warn("could not read the memories a conversation's replies used", zap.Error(err))

		return
	}

	notes := make(map[pulid.ID]conversation.MemoryNote, len(found))
	for _, memory := range found {
		notes[memory.Memory.ID] = memoryNote(memory)
	}
	for idx := range messages {
		message := &messages[idx]
		for _, id := range messageMemoryIDs(message) {
			if note, ok := notes[id]; ok {
				message.Memories = append(message.Memories, note)
			}
		}
	}
}

// deskMemoryActor is the reader with the permission that decides whether
// they may change a memory their team or organization shares.
func (s *Describer) deskMemoryActor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) *agentmemoryports.DeskMemoryActor {
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}

	return &agentmemoryports.DeskMemoryActor{
		Actor:           actor,
		MayManageShared: s.Memories.MayManageShared(ctx, tenant, userID),
	}
}
