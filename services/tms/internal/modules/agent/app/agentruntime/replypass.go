package agentruntime

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/recordids"
)

// passReply holds a final reply to two rules the prompt already gives, in
// code beneath them: a person never reads an internal record id, and a table
// kept beside the conversation is pointed to, not written out again. Models
// break both, small ones most, and the prompt alone left a reply full of
// shp_01J… and a twenty-five row markdown copy of the table next to it.
//
// The reply has already streamed. When the pass changes it, a reply_replaced
// event puts the corrected reply in place of the streamed one, so the person
// reads what is recorded without the turn appearing to fail and start over.
// Each pass that changed something is a reply_regrounded event in the
// trajectory.
func (s *Service) passReply(
	t *Turn,
	fx TurnEffects,
	completion *llmports.ChatCompletionResult,
) {
	if completion == nil || strings.TrimSpace(completion.Text) == "" {
		return
	}

	stripped := recordids.Strip(completion.Text)
	pointed, table := replycheck.PointToTables(stripped, replycheck.KeptTables(t))
	if pointed == completion.Text {
		return
	}

	if stripped != completion.Text {
		fx.Emit(
			replycheck.PassEvent(assistantports.RegroundStripIDs, replycheck.StripIDsReason, nil),
		)
	}
	if table != nil {
		fx.Emit(
			replycheck.PassEvent(
				assistantports.RegroundPointToTable,
				replycheck.PointToTableReason,
				table,
			),
		)
	}
	fx.Emit(replycheck.ReplyReplacedEvent(pointed, replycheck.ReplyPassReplacedReason))
	completion.Text = pointed
}

// keepShown remembers what the call put beside the conversation, so the
// reply passes can check the reply names it and point a reprint to it.
func (t *Turn) keepShown(outcome *toolcall.Result) {
	if outcome.Failed || outcome.Shown == nil {
		return
	}
	for idx := range t.artifacts {
		if outcome.Shown.Supersedes(&t.artifacts[idx]) {
			t.artifacts[idx] = *outcome.Shown

			return
		}
	}
	t.artifacts = append(t.artifacts, *outcome.Shown)
}

func (t *Turn) foldReplaced(table *agentruntimeports.ShownArtifact) {
	if table == nil || len(table.Replaces) == 0 {
		return
	}
	unread := 0
	for idx := len(t.messages) - 1; idx >= 0; idx-- {
		if t.messages[idx].Role == llmports.RoleAssistant {
			unread = idx + 1
			break
		}
	}
	folded := runtimetools.FoldedNote(table)
	for _, cardID := range table.Replaces {
		idx := slices.IndexFunc(t.artifacts, func(shown agentruntimeports.ShownArtifact) bool {
			return shown.ID == cardID
		})
		if idx < 0 {
			continue
		}
		stale := runtimetools.ShownNote(&t.artifacts[idx])
		t.artifacts = slices.Delete(t.artifacts, idx, idx+1)
		for msgIdx := unread; msgIdx < len(t.messages); msgIdx++ {
			msg := &t.messages[msgIdx]
			if msg.Role != llmports.RoleTool || !strings.HasSuffix(msg.Content, stale) {
				continue
			}
			msg.Content = strings.TrimSuffix(msg.Content, stale) + folded
			replycheck.Refold(t, msg.ToolCallID, stale, folded)
		}
	}
}
