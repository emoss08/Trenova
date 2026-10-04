package assistantservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

// keepTurnMemories puts what the turn did with memory on its last reply of
// its own: the memories it used, shown above the reply, and what it kept,
// shown under it. Another agent's steps are never that reply.
func keepTurnMemories(messages []conversation.Message, run *serviceports.RunResult) {
	if run == nil || (len(run.UsedMemoryIDs) == 0 && len(run.SavedMemories) == 0) {
		return
	}

	for idx := len(messages) - 1; idx >= 0; idx-- {
		message := &messages[idx]
		if message.Role != conversation.RoleAssistant || message.AgentDefinitionID.IsNotNil() {
			continue
		}
		message.UsedMemoryIDs = slices.Clone(run.UsedMemoryIDs)
		message.SavedMemories = slices.Clone(run.SavedMemories)

		return
	}
}

// describeMemories fills each reply's memories as its reader may see them,
// in one read for the whole page. A memory since moved out of the reader's
// reach is left out, and a reply whose memories cannot be read is served
// without them rather than not at all.
func (s *Service) describeMemories(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
	messages []conversation.Message,
) {
	if s.memories == nil || userID.IsNil() {
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
	found, err := s.memories.DeskByIDs(ctx, actor, sliceutils.Dedupe(ids))
	if err != nil {
		s.logger.Warn("could not read the memories a conversation's replies used", zap.Error(err))

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

func messageMemoryIDs(message *conversation.Message) []pulid.ID {
	if len(message.UsedMemoryIDs) == 0 && len(message.SavedMemories) == 0 {
		return nil
	}

	ids := slices.Clone(message.UsedMemoryIDs)
	for _, saved := range message.SavedMemories {
		if !slices.Contains(ids, saved.ID) {
			ids = append(ids, saved.ID)
		}
	}

	return ids
}

// deskMemoryActor is the reader with the permissions that decide whether
// they may change a memory their team or organization shares.
func (s *Service) deskMemoryActor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) *serviceports.DeskMemoryActor {
	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}

	return &serviceports.DeskMemoryActor{
		Actor:           actor,
		MayCreateShared: s.allowed(ctx, actor, permission.OpCreate),
		MayUpdateShared: s.allowed(ctx, actor, permission.OpUpdate),
	}
}

func (s *Service) allowed(
	ctx context.Context,
	actor *serviceports.RequestActor,
	operation permission.Operation,
) bool {
	if s.permissions == nil {
		return false
	}

	result, err := s.permissions.Check(ctx, actor.PermissionCheck(permission.ResourceAgentMemory, operation))

	return err == nil && result != nil && result.Allowed
}

func memoryNote(memory *serviceports.DeskMemory) conversation.MemoryNote {
	note := conversation.MemoryNote{
		ID:          memory.Memory.ID,
		Content:     memory.Memory.Content,
		Kind:        string(memory.Memory.Kind),
		Scope:       string(memory.Memory.Scope),
		RoleName:    memory.RoleName,
		Status:      string(memory.Memory.Status),
		Source:      string(memory.Memory.Source),
		SourceTitle: memory.SourceTitle,
		CreatedAt:   memory.Memory.CreatedAt,
		Version:     memory.Memory.Version,
		Editable:    memory.Editable,
	}
	if memory.Memory.RoleID != nil {
		note.RoleID = *memory.Memory.RoleID
	}

	return note
}
