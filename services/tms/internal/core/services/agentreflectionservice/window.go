package agentreflectionservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/agentmemoryservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const maxWindowRecords = 40

func (s *Service) threadProposals(
	ctx context.Context,
	tenant pagination.TenantInfo,
	threadID pulid.ID,
	messages []conversation.Message,
) []DecidedProposal {
	stored, err := s.proposals.ListByThread(ctx, repositories.ListAgentProposalsByThreadRequest{
		ThreadID:   threadID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.l.Warn("agent reflection: the conversation's proposals could not be read",
			zap.String("thread", threadID.String()),
			zap.Error(err),
		)

		return nil
	}

	inWindow := make(map[pulid.ID]struct{}, len(messages))
	for idx := range messages {
		inWindow[messages[idx].ID] = struct{}{}
	}
	since := messages[0].CreatedAt

	return s.decided(
		ctx,
		tenant,
		stored,
		func(proposal *agent.AgentProposal, decision *agent.AgentDecision) bool {
			if _, ok := inWindow[proposal.SourceMessageID]; ok {
				return true
			}

			return decision.CreatedAt >= since
		},
	)
}

func (s *Service) runProposals(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
) []DecidedProposal {
	stored, err := s.proposals.ListByRun(ctx, repositories.ListAgentProposalsByRunRequest{
		RunID:      runID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.l.Warn("agent reflection: the run's proposals could not be read",
			zap.String("run", runID.String()),
			zap.Error(err),
		)

		return nil
	}

	return s.decided(ctx, tenant, stored, func(*agent.AgentProposal, *agent.AgentDecision) bool {
		return true
	})
}

func (s *Service) decided(
	ctx context.Context,
	tenant pagination.TenantInfo,
	stored []*agent.AgentProposal,
	keep func(*agent.AgentProposal, *agent.AgentDecision) bool,
) []DecidedProposal {
	ids := make([]pulid.ID, 0, len(stored))
	byID := make(map[pulid.ID]*agent.AgentProposal, len(stored))
	for _, proposal := range stored {
		if proposal != nil && proposal.Status != agent.ProposalStatusPending {
			ids = append(ids, proposal.ID)
			byID[proposal.ID] = proposal
		}
	}
	if len(ids) == 0 {
		return nil
	}

	decisions, err := s.decisions.ListByProposals(
		ctx,
		repositories.ListAgentDecisionsByProposalsRequest{
			ProposalIDs: ids,
			TenantInfo:  tenant,
		},
	)
	if err != nil {
		s.l.Warn("agent reflection: the decisions on proposals could not be read", zap.Error(err))

		return nil
	}

	out := make([]DecidedProposal, 0, len(decisions))
	for _, decision := range decisions {
		if decision == nil || decision.ProposalID == nil {
			continue
		}
		proposal, ok := byID[*decision.ProposalID]
		if !ok || !keep(proposal, decision) {
			continue
		}
		outcome := agent.OutcomeOfDecision(decision.Decision, decision.Modifications)
		if outcome != agent.TrustOutcomeModified && outcome != agent.TrustOutcomeRejected {
			continue
		}
		reason := strings.TrimSpace(decision.Note)
		if reason == "" {
			reason = agentmemoryservice.HumanReason(decision.ReasonCode)
		}
		out = append(out, DecidedProposal{
			ToolName: proposal.ToolName,
			Outcome:  outcome,
			Reason:   reason,
		})
	}

	return out
}

func (s *Service) negativeRatings(
	ctx context.Context,
	tenant pagination.TenantInfo,
	person pulid.ID,
	messages []conversation.Message,
) int {
	if s.feedback == nil || person.IsNil() {
		return 0
	}

	targets := make([]repositories.AIFeedbackTargetRef, 0, len(messages))
	for idx := range messages {
		if messages[idx].Role == conversation.RoleAssistant {
			targets = append(targets, repositories.AIFeedbackTargetRef{
				TargetType: aifeedback.TargetAssistantMessage,
				TargetID:   messages[idx].ID,
			})
		}
	}
	if len(targets) == 0 {
		return 0
	}

	ratings, err := s.feedback.ListForTargets(ctx, repositories.ListAIFeedbackForTargetsRequest{
		TenantInfo: tenant,
		UserID:     person,
		Targets:    targets,
	})
	if err != nil {
		s.l.Warn("agent reflection: the ratings of the conversation's replies could not be read",
			zap.Error(err),
		)

		return 0
	}

	negative := 0
	for _, rating := range ratings {
		if rating != nil && rating.Rating == aifeedback.RatingNegative {
			negative++
		}
	}

	return negative
}

func (s *Service) runMessages(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
) ([]conversation.Message, error) {
	runs, err := s.runs.ListTranscriptsByIDs(ctx, repositories.ListAgentRunsByIDsRequest{
		IDs:        []pulid.ID{runID},
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("read the run's transcript: %w", err)
	}

	for _, run := range runs {
		if run == nil || run.ID != runID || run.Transcript == nil {
			continue
		}
		messages := make([]conversation.Message, 0, len(run.Transcript.Messages))
		for idx := range run.Transcript.Messages {
			messages = append(
				messages,
				conversation.MessageOfTranscript(&run.Transcript.Messages[idx]),
			)
		}

		return messages, nil
	}

	return nil, nil
}

func windowRecords(messages []conversation.Message) []agent.EntityRef {
	records := make([]agent.EntityRef, 0, maxWindowRecords)
	seen := make(map[agent.EntityRef]struct{}, maxWindowRecords)
	add := func(kind, id string) {
		if len(records) >= maxWindowRecords || kind == "" || id == "" {
			return
		}
		ref := agent.EntityRef{Type: kind, ID: id}
		if _, ok := seen[ref]; ok {
			return
		}
		seen[ref] = struct{}{}
		records = append(records, ref)
	}
	addIDs := func(text string) {
		for _, id := range recordIDsIn(text) {
			if kind, ok := agent.MemoryRecordKindOfID(id); ok {
				add(string(kind), id.String())
			}
		}
	}

	for idx := range messages {
		message := &messages[idx]
		for _, mention := range message.Mentions {
			add(mention.Type, mention.ID)
		}
		if message.PageContext != nil {
			add(message.PageContext.EntityType, message.PageContext.EntityID)
		}
		for _, call := range message.ToolCalls {
			addIDs(callArguments(call.Arguments))
		}
		if message.Role == conversation.RoleTool {
			addIDs(message.Content)
		}
	}

	return records
}
