package memorynotes

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
)

// KeepTurnMemories puts what the turn did with memory on its last reply of
// its own: the memories it used, shown above the reply, and what it kept,
// shown under it. Another agent's steps are never that reply.
func KeepTurnMemories(messages []conversation.Message, run *agentruntimeports.RunResult) {
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

func memoryNote(memory *agentmemoryports.DeskMemory) conversation.MemoryNote {
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
		Reason:      memory.Memory.Evidence.GetReason(),
		Replaces:    memoryNoteLink(memory.Replaces),
		ReplacedBy:  memoryNoteLink(memory.ReplacedBy),
	}
	if memory.Memory.RoleID != nil {
		note.RoleID = *memory.Memory.RoleID
	}

	return note
}

func memoryNoteLink(link *agentmemoryports.DeskMemoryLink) *conversation.MemoryNoteLink {
	if link == nil {
		return nil
	}

	return &conversation.MemoryNoteLink{
		ID:      link.ID,
		Content: link.Content,
		Status:  string(link.Status),
	}
}
