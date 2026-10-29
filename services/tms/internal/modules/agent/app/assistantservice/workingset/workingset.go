package workingset

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationworkingset"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Turn struct {
	Thread    *conversation.Thread
	Page      *agentpagecontext.PageContext
	Mentions  []agentpagecontext.EntityRef
	Proposals []agentruntimeports.ProposalOutcome
	Saved     []conversation.Message
	Actions   []agentruntimeports.PendingAction
	At        int64
}

func TouchedRecords(turn *Turn) []conversation.WorkingRecord {
	touched := make([]conversation.WorkingRecord, 0, 8)
	add := func(id, label string, source conversation.WorkingSource) {
		if record, ok := conversationworkingset.NewWorkingRecord(id, label, source, turn.At); ok {
			touched = append(touched, record)
		}
	}

	if turn.Thread.HasSubject() {
		add(turn.Thread.SubjectID.String(), "", conversation.WorkingSourceSubject)
	}
	if turn.Page != nil {
		add(turn.Page.EntityID, turn.Page.Title, conversation.WorkingSourcePage)
	}
	for _, mention := range turn.Mentions {
		add(mention.ID, mention.Label, conversation.WorkingSourceMention)
	}

	failed := make(map[string]bool, len(turn.Saved))
	summaries := make(map[string]string, len(turn.Saved))
	for idx := range turn.Saved {
		message := &turn.Saved[idx]
		if message.Role == conversation.RoleTool && message.ToolCallID != "" {
			failed[message.ToolCallID] = message.ToolFailed
			summaries[message.ToolCallID] = message.ToolSummary
		}
	}
	for idx := range turn.Saved {
		message := &turn.Saved[idx]
		if message.Role != conversation.RoleAssistant || message.Delegated() {
			continue
		}
		for _, call := range message.ToolCalls {
			if failed[call.ID] {
				continue
			}
			for _, record := range worldwatch.RecordsRead(call.Name, call.Arguments, summaries[call.ID]) {
				add(record.ID, record.Label, conversation.WorkingSourceRead)
			}
		}
	}

	for idx := range turn.Actions {
		action := &turn.Actions[idx]
		if !action.Executed || action.ExecutionError != "" {
			continue
		}
		if action.Target != nil && action.Target.ID.IsNotNil() {
			add(action.Target.ID.String(), "", conversation.WorkingSourceWrote)
		}
		if id, label := madeRecord(action.ExecutionResult); id != "" {
			add(id, label, conversation.WorkingSourceWrote)
		}
	}

	since := lastTouched(turn.Thread.WorkingSet)
	for idx := range turn.Proposals {
		outcome := &turn.Proposals[idx]
		if outcome.Status != agent.ProposalStatusExecuted || outcome.ExecutedAt == nil ||
			*outcome.ExecutedAt < since {
			continue
		}
		if id, label := madeRecord(outcome.ExecutionResult); id != "" {
			add(id, label, conversation.WorkingSourceWrote)
		}
	}

	return touched
}

// madeRecord is the one record an executed write made or changed, so a load
// the person approved into being is what the conversation is about next,
// though no turn ran the write.
func madeRecord(result *agentrecord.ToolExecutionResult) (id, label string) {
	if result == nil || result.Record == nil || result.Record.ID == "" {
		return "", ""
	}

	return result.Record.ID, result.Name
}

func lastTouched(set []conversation.WorkingRecord) int64 {
	var latest int64
	for idx := range set {
		latest = max(latest, set[idx].TouchedAt)
	}

	return latest
}

type AnchorScope struct {
	Thread   *conversation.Thread
	Page     *agentpagecontext.PageContext
	Mentions []agentpagecontext.EntityRef
	Actor    *permissionports.RequestActor
	Tenant   pagination.TenantInfo
}

func AnchorRefs(scope *AnchorScope) []agentpagecontext.EntityRef {
	refs := make(
		[]agentpagecontext.EntityRef,
		0,
		len(scope.Mentions)+len(scope.Thread.WorkingSet)+1,
	)
	subject := ""
	if scope.Thread.HasSubject() {
		subject = scope.Thread.SubjectID.String()
	}
	add := func(ref agentpagecontext.EntityRef) {
		if ref.ID == "" || ref.ID == subject ||
			slices.ContainsFunc(
				refs,
				func(known agentpagecontext.EntityRef) bool { return known.ID == ref.ID },
			) {
			return
		}
		refs = append(refs, ref)
	}

	for _, mention := range scope.Mentions {
		add(
			agentpagecontext.EntityRef{
				Type:  mention.Type,
				ID:    strings.TrimSpace(mention.ID),
				Label: mention.Label,
			},
		)
	}
	if scope.Page != nil {
		add(agentpagecontext.EntityRef{
			Type:  scope.Page.EntityType,
			ID:    strings.TrimSpace(scope.Page.EntityID),
			Label: scope.Page.Title,
		})
	}
	for _, record := range scope.Thread.WorkingSet {
		add(record.Ref())
	}
	if len(refs) > conversationworkingset.MaxWorkingSet {
		refs = refs[:conversationworkingset.MaxWorkingSet]
	}

	return refs
}
