package assistantservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type workingTurn struct {
	thread  *conversation.Thread
	plan    *TurnPlan
	saved   []conversation.Message
	actions []services.PendingAction
	at      int64
}

func touchedRecords(turn *workingTurn) []conversation.WorkingRecord {
	touched := make([]conversation.WorkingRecord, 0, 8)
	add := func(id, label string, source conversation.WorkingSource) {
		if record, ok := conversation.NewWorkingRecord(id, label, source, turn.at); ok {
			touched = append(touched, record)
		}
	}

	if turn.thread.HasSubject() {
		add(turn.thread.SubjectID.String(), "", conversation.WorkingSourceSubject)
	}
	if turn.plan != nil {
		if turn.plan.Page != nil {
			add(turn.plan.Page.EntityID, turn.plan.Page.Title, conversation.WorkingSourcePage)
		}
		for _, mention := range turn.plan.Mentions {
			add(mention.ID, mention.Label, conversation.WorkingSourceMention)
		}
	}

	failed := make(map[string]bool, len(turn.saved))
	summaries := make(map[string]string, len(turn.saved))
	for idx := range turn.saved {
		message := &turn.saved[idx]
		if message.Role == conversation.RoleTool && message.ToolCallID != "" {
			failed[message.ToolCallID] = message.ToolFailed
			summaries[message.ToolCallID] = message.ToolSummary
		}
	}
	for idx := range turn.saved {
		message := &turn.saved[idx]
		if message.Role != conversation.RoleAssistant || message.Delegated() {
			continue
		}
		for _, call := range message.ToolCalls {
			if failed[call.ID] {
				continue
			}
			for _, record := range agentruntime.RecordsRead(call.Name, call.Arguments, summaries[call.ID]) {
				add(record.ID, record.Label, conversation.WorkingSourceRead)
			}
		}
	}

	for idx := range turn.actions {
		action := &turn.actions[idx]
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

	if turn.plan != nil {
		since := lastTouched(turn.thread.WorkingSet)
		for idx := range turn.plan.Proposals {
			outcome := &turn.plan.Proposals[idx]
			if outcome.Status != agent.ProposalStatusExecuted || outcome.ExecutedAt == nil ||
				*outcome.ExecutedAt < since {
				continue
			}
			if id, label := madeRecord(outcome.ExecutionResult); id != "" {
				add(id, label, conversation.WorkingSourceWrote)
			}
		}
	}

	return touched
}

// madeRecord is the one record an executed write made or changed, so a load
// the person approved into being is what the conversation is about next,
// though no turn ran the write.
func madeRecord(result *agent.ToolExecutionResult) (id, label string) {
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

func (s *Service) keepWorkingSet(
	ctx context.Context,
	turn *workingTurn,
	tenant pagination.TenantInfo,
) {
	touched := touchedRecords(turn)
	if len(touched) == 0 {
		return
	}

	next := conversation.Touch(turn.thread.WorkingSet, touched...)
	if err := s.conversations.UpdateThreadContext(ctx, repositories.UpdateThreadContextRequest{
		ThreadID:   turn.thread.ID,
		TenantInfo: tenant,
		WorkingSet: &next,
	}); err != nil {
		s.logger.Warn("could not keep the records a conversation is about",
			zap.String("thread", turn.thread.ID.String()),
			zap.Error(err),
		)
		return
	}
	turn.thread.WorkingSet = next
}

type anchorScope struct {
	thread   *conversation.Thread
	page     *agent.PageContext
	mentions []agent.EntityRef
	actor    *services.RequestActor
	tenant   pagination.TenantInfo
}

func anchorRefs(scope *anchorScope) []agent.EntityRef {
	refs := make([]agent.EntityRef, 0, len(scope.mentions)+len(scope.thread.WorkingSet)+1)
	subject := ""
	if scope.thread.HasSubject() {
		subject = scope.thread.SubjectID.String()
	}
	add := func(ref agent.EntityRef) {
		if ref.ID == "" || ref.ID == subject ||
			slices.ContainsFunc(refs, func(known agent.EntityRef) bool { return known.ID == ref.ID }) {
			return
		}
		refs = append(refs, ref)
	}

	for _, mention := range scope.mentions {
		add(agent.EntityRef{Type: mention.Type, ID: strings.TrimSpace(mention.ID), Label: mention.Label})
	}
	if scope.page != nil {
		add(agent.EntityRef{
			Type: scope.page.EntityType, ID: strings.TrimSpace(scope.page.EntityID), Label: scope.page.Title,
		})
	}
	for _, record := range scope.thread.WorkingSet {
		add(record.Ref())
	}
	if len(refs) > conversation.MaxWorkingSet {
		refs = refs[:conversation.MaxWorkingSet]
	}

	return refs
}

func (s *Service) anchorRecords(
	ctx context.Context,
	scope *anchorScope,
) []agentdefinition.RuntimeAnchor {
	if s.anchors == nil {
		return nil
	}
	refs := anchorRefs(scope)
	if len(refs) == 0 {
		return nil
	}

	return s.anchors.Anchor(ctx, &services.RecordAnchorRequest{
		TenantInfo: scope.tenant,
		Actor:      scope.actor,
		Records:    refs,
	})
}
