package turnmemory

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

// Announce tells the reader which memories bore on the turn, before
// the first word of the reply, so the note that says so sits above it.
func Announce(t Turn, fx Emitter) {
	if len(t.Result().UsedMemoryIDs) == 0 {
		return
	}

	fx.Emit(memoryUsedEvent(t.Result().UsedMemoryIDs))
}

// Note keeps what a call did with memory: the memories a recall read
// back join those the turn used, and a memory a remember kept or offered is
// announced so the reply can show it.
func Note(t Turn, fx Emitter, outcome *toolcall.Result) {
	if t.Request().Delegation != nil {
		return
	}

	added := false
	for _, id := range outcome.Memories {
		if id.IsNotNil() && !slices.Contains(t.Result().UsedMemoryIDs, id) {
			t.Result().UsedMemoryIDs = append(t.Result().UsedMemoryIDs, id)
			added = true
		}
	}
	if added {
		fx.Emit(memoryUsedEvent(t.Result().UsedMemoryIDs))
	}

	if saved := outcome.Saved; saved != nil && saved.ID.IsNotNil() {
		t.Result().SavedMemories = append(t.Result().SavedMemories, *saved)
		fx.Emit(kernelports.StreamEvent{
			Event: assistantstreamports.AssistantEventMemorySaved,
			Data:  *saved,
		})
	}
}

// NoteTool counts the memories about a tool as used once the turn
// calls it: they were carried because the tool was on hand, and bear on the
// reply only when it is.
func NoteTool(t Turn, fx Emitter, tool string) {
	if t.Request().Delegation != nil || len(t.ToolMemories()) == 0 {
		return
	}
	ids, ok := t.ToolMemories()[tool]
	if !ok {
		return
	}
	delete(t.ToolMemories(), tool)

	added := false
	for _, id := range ids {
		if id.IsNotNil() && !slices.Contains(t.Result().UsedMemoryIDs, id) {
			t.Result().UsedMemoryIDs = append(t.Result().UsedMemoryIDs, id)
			added = true
		}
	}
	if added {
		fx.Emit(memoryUsedEvent(t.Result().UsedMemoryIDs))
	}
}

func memoryUsedEvent(ids []pulid.ID) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventMemoryUsed,
		Data:  assistantstreamports.AssistantMemoryUsedEvent{IDs: slices.Clone(ids)},
	}
}
