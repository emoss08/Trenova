package assistantcompaction

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/agentaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/chaptertitles"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/contextusage"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposaloutcomes"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnlimits"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationreplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// PrepareCompaction reads a conversation and makes ready the request that
// summarizes everything before its latest turns.
//
// It refuses what a turn would refuse, since a compaction is a model call on
// the agent's budget and the person's behalf: a conversation that can no
// longer continue, an agent switched off, a spent budget. It also refuses a
// conversation with nothing worth compacting.
func (s *Service) PrepareCompaction(
	ctx context.Context,
	req *CompactRequest,
	actor *permissionports.RequestActor,
) (*CompactionPlan, error) {
	thread, err := s.conversations.GetThread(ctx, &conversationports.GetThreadRequest{
		ID:         req.ThreadID,
		UserID:     actor.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	definition, err := s.access().
		UsableDefinition(ctx, thread, actor, &assistantports.SendMessageRequest{
			ThreadID:   thread.ID,
			TenantInfo: req.TenantInfo,
		})
	if err != nil {
		return nil, err
	}
	if err = s.limits().AssertWithinBudget(ctx, definition); err != nil {
		return nil, err
	}

	history, err := s.usage().ModelHistory(ctx, thread.ID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	older, through := turnhistory.SplitForCompaction(history)
	if len(older) == 0 {
		return nil, errNothingToCompact()
	}
	titles := s.chapterList().Titles(ctx, thread.ID, req.TenantInfo)
	older, through = chaptertitles.SnapToChapter(older, through, titles)

	outcomes := s.outcomes().Outcomes(ctx, thread, req.TenantInfo)
	before := contextusage.CurrentUsage(thread, history, outcomes)
	if !before.WorthCompacting() {
		return nil, errNothingToCompact()
	}

	kept := []string{conversationreplay.KeptRecent}
	if len(proposaloutcomes.Pending(outcomes)) > 0 {
		kept = append(kept, conversationreplay.KeptApprovals)
	}

	return &CompactionPlan{
		ThreadID:     thread.ID,
		Auto:         req.Auto,
		Through:      through,
		Summarized:   summarizedCount(older),
		Before:       before.Total(),
		After:        before.Total() - before.Frees(),
		Kept:         kept,
		Instructions: before.Instructions,
		Request: &llmports.ChatCompletionRequest{
			TenantInfo: actor.TenantInfo(),
			System:     compactionPrompt(definition.Name, chaptered(older, titles)),
			Messages: []llmports.Message{
				{Role: llmports.RoleUser, Content: compactionTranscript(older, titles)},
			},
			MaxTokens:           conversation.SummaryTokenBudget,
			PreferredProviderID: compactionProvider(thread, definition),
			Attribution:         compactionAttribution(definition, thread, actor, req.TurnID),
		},
	}, nil
}

// FinishCompaction saves the summary a compaction produced and measures the
// conversation after it.
//
// It is safe to run twice for one plan: a summary already saved for the
// same stretch is returned rather than saved again, so an activity retried
// after its save landed does not leave two.
func (s *Service) FinishCompaction(
	ctx context.Context,
	plan *CompactionPlan,
	reply *CompactionReply,
	actor *permissionports.RequestActor,
) (*CompactionResult, error) {
	summary := strings.TrimSpace(reply.Summary)
	if summary == "" {
		return nil, errortypes.NewBusinessError(
			"The model returned no summary, so nothing was compacted. Try again in a moment.",
		)
	}
	summary = stringutils.Ellipsize(summary, maxSummaryChars)

	tenant := actor.TenantInfo()
	thread, err := s.conversations.GetThread(ctx, &conversationports.GetThreadRequest{
		ID:         plan.ThreadID,
		UserID:     actor.UserID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	history, err := s.usage().ModelHistory(ctx, thread.ID, tenant)
	if err != nil {
		return nil, err
	}
	if saved := savedCompaction(history, plan.Through); saved != nil {
		return &CompactionResult{Message: saved, Usage: thread.ContextUsage}, nil
	}

	message := conversation.Message{
		Role:         conversation.RoleUser,
		Kind:         conversation.MessageKindCompaction,
		Content:      summary,
		Model:        reply.Model,
		ProviderID:   reply.ProviderID,
		InputTokens:  reply.Input,
		OutputTokens: reply.Output,
		Compaction: &conversation.Compaction{
			Auto:       plan.Auto,
			Summarized: plan.Summarized,
			Through:    plan.Through,
			Before:     plan.Before,
			Kept:       plan.Kept,
		},
	}

	// The context after is measured on the history as it will read, with
	// the summary numbered after everything already in it.
	next := len(history)
	if next > 0 {
		next = history[len(history)-1].Sequence + 1
	}
	pending := message
	pending.Sequence = next
	usage := turnhistory.MeasureContext(turnhistory.ContextRequest{
		Instructions: plan.Instructions,
		History:      append(history, pending),
		Proposals:    s.outcomes().Outcomes(ctx, thread, tenant),
		Model:        contextusage.UsageModel(thread, reply.Model),
		Window:       contextusage.UsageWindow(thread, reply.Model, reply.ContextWindow),
		Now:          timeutils.NowUnix(),
	})
	message.Compaction.After = usage.Total()

	saved, err := s.conversations.AppendTurn(ctx, &conversationports.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
		Messages:   []conversation.Message{message},
	})
	if err != nil {
		return nil, fmt.Errorf("save the compaction summary: %w", err)
	}
	if len(saved) == 0 {
		return nil, fmt.Errorf("save the compaction summary: nothing was saved")
	}

	s.usage().KeepContextUsage(ctx, thread, &usage, tenant)

	return &CompactionResult{Message: &saved[0], Usage: &usage}, nil
}

// StopCompactingItself turns off a conversation's compacting itself, which is
// what cancelling a compaction that started on its own means: the person
// would rather the conversation stayed as it is.
func (s *Service) StopCompactingItself(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
) error {
	off := true

	return s.conversations.UpdateThreadContext(ctx, &conversationports.UpdateThreadContextRequest{
		ThreadID:       threadID,
		TenantInfo:     tenant,
		AutoCompactOff: &off,
	})
}

type Service struct {
	logger        *zap.Logger
	conversations conversationports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	permissions   permissionports.PermissionEngine
	queries       agenttoolports.AgentQueryToolRegistry
	budgets       agentbudgetports.AgentBudgetService
	agentControls tenantports.AgentControlRepository
	quota         platformplanports.QuotaGuard
	chapters      assistantchapterports.ChapterRepository
	proposals     proposalstores.ProposalStore
	decisions     agentdecisionports.AgentDecisionRepository
}

func (s *Service) access() *agentaccess.Access {
	return &agentaccess.Access{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Queries:     s.queries,
	}
}

func (s *Service) limits() *turnlimits.Limits {
	return &turnlimits.Limits{
		Budgets:       s.budgets,
		Conversations: s.conversations,
		AgentControls: s.agentControls,
		Quota:         s.quota,
	}
}

func (s *Service) chapterList() *chaptertitles.Index {
	return &chaptertitles.Index{Logger: s.logger, Chapters: s.chapters}
}

func (s *Service) outcomes() *proposaloutcomes.Reader {
	return &proposaloutcomes.Reader{
		Logger:    s.logger,
		Proposals: s.proposals,
		Decisions: s.decisions,
	}
}

func (s *Service) usage() *contextusage.Usage {
	return &contextusage.Usage{Logger: s.logger, Conversations: s.conversations}
}
