package agentevalcaseservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

type frozenTurn struct {
	threadID  pulid.ID
	messageID pulid.ID
	input     string
	history   []agentquality.HistoryMessage
	page      *agent.PageContext
	mentions  []agent.EntityRef
	fixtures  []agentquality.ToolFixture
	called    []string
}

type turnAnchor struct {
	threadID  pulid.ID
	messageID pulid.ID
	before    int64
}

func (s *Service) CreateFromProposal(
	ctx context.Context,
	req *services.CreateEvalCaseFromProposalRequest,
	actor *services.RequestActor,
) (*services.EvalCaseCapture, error) {
	tenantInfo := req.TenantInfo
	existing, err := s.cases.GetByProposal(ctx, repositories.GetAgentEvalCaseByProposalRequest{
		ProposalID: req.ProposalID,
		TenantInfo: tenantInfo,
	})
	switch {
	case err == nil:
		return &services.EvalCaseCapture{Case: existing, Duplicate: true}, nil
	case !isNotFound(err):
		return nil, err
	}

	proposal, err := s.proposals.GetByID(ctx, repositories.GetAgentProposalByIDRequest{
		ID:         req.ProposalID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	latest, err := s.latestDecision(ctx, tenantInfo, proposal.ID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return nil, errortypes.NewBusinessError(
			"Only a proposal a person approved or rejected can become an evaluation case",
		)
	}

	run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         proposal.RunID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if run.AgentDefinitionID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"This proposal was not made by a configured agent, so there is nothing to evaluate",
		)
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         run.AgentDefinitionID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	original := agent.NewOriginalProposal(proposal, latest)
	rejected := latest.Decision == agent.DecisionRejected
	runID := run.ID
	proposalID := proposal.ID
	evalCase := &agentquality.EvalCase{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		Title:             strings.TrimSpace(req.Title),
		Source:            agentquality.CaseSourceDecidedProposal,
		Status:            agentquality.CaseStatusCandidate,
		Trigger:           run.Trigger,
		SourceRunID:       &runID,
		SourceProposalID:  &proposalID,
		HeldTools:         definition.EffectiveToolNames(),
		Expected: agentquality.Expected{
			ToolMode: agentquality.ToolMatchAnyOrder,
			Proposals: []agentquality.ExpectedProposal{{
				ToolName:         proposal.ToolName,
				Params:           original.Expected(),
				Rejected:         rejected,
				SourceProposalID: proposal.ID.String(),
			}},
		},
		CapturedFingerprint: agentquality.FingerprintOf(definition, promptVersion),
	}
	if evalCase.Title == "" {
		evalCase.Title = proposalTitle(proposal.ToolName, rejected)
	}

	if run.SubjectType == agent.SubjectAssistantThread {
		frozen, redacted, freezeErr := s.freezeChat(ctx, tenantInfo, turnAnchor{
			threadID:  run.SubjectID,
			messageID: proposal.SourceMessageID,
			before:    run.CreatedAt,
		})
		if freezeErr != nil {
			return nil, freezeErr
		}
		applyFrozen(evalCase, frozen, redacted, s.now())
	} else {
		evalCase.Input = s.backgroundInput(ctx, tenantInfo, run)
		evalCase.SubjectType = run.SubjectType
		evalCase.SubjectID = run.SubjectID
	}

	return s.save(ctx, evalCase, actor)
}

func (s *Service) CreateFromMessage(
	ctx context.Context,
	req *services.CreateEvalCaseFromMessageRequest,
	actor *services.RequestActor,
) (*services.EvalCaseCapture, error) {
	tenantInfo := req.TenantInfo
	thread, err := s.conversations.GetThreadOwned(ctx, repositories.GetThreadOwnedRequest{
		ID:         req.ThreadID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	frozen, redacted, err := s.freezeChat(ctx, tenantInfo, turnAnchor{
		threadID:  thread.ID,
		messageID: req.MessageID,
	})
	if err != nil {
		return nil, err
	}
	if frozen.messageID != req.MessageID {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalid,
			"Only a reply the assistant wrote in this conversation can become a case",
		)
	}

	tools := make([]agentquality.ExpectedTool, 0, len(frozen.called))
	for _, name := range frozen.called {
		tools = append(tools, agentquality.ExpectedTool{Name: name})
	}

	evalCase := &agentquality.EvalCase{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		Title:             strings.TrimSpace(req.Title),
		Source:            agentquality.CaseSourceThumbsUp,
		Status:            agentquality.CaseStatusCandidate,
		Trigger:           agent.RunTriggerChat,
		HeldTools:         definition.EffectiveToolNames(),
		Expected: agentquality.Expected{
			ToolMode: agentquality.ToolMatchAnyOrder,
			Tools:    tools,
		},
		CapturedFingerprint: agentquality.FingerprintOf(definition, promptVersion),
	}
	if req.FeedbackID.IsNotNil() {
		feedbackID := req.FeedbackID
		evalCase.SourceFeedbackID = &feedbackID
	}
	applyFrozen(evalCase, frozen, redacted, s.now())
	if evalCase.Title == "" {
		evalCase.Title = titleFromInput(evalCase.Input)
	}

	return s.save(ctx, evalCase, actor)
}

func (s *Service) CaptureCandidates(
	ctx context.Context,
	req services.CaptureEvalCaseCandidatesRequest,
) (*services.CaptureEvalCaseCandidatesResult, error) {
	candidates, err := s.cases.ListCaptureCandidates(
		ctx,
		repositories.ListEvalCaseCaptureCandidatesRequest{Since: req.Since, Limit: req.Limit},
	)
	if err != nil {
		return nil, err
	}

	result := &services.CaptureEvalCaseCandidatesResult{Scanned: len(candidates)}
	for _, candidate := range candidates {
		request := &services.CreateEvalCaseFromProposalRequest{
			ProposalID: candidate.ProposalID,
			TenantInfo: candidate.TenantInfo(),
		}
		captured, captureErr := s.CreateFromProposal(ctx, request, nil)
		switch {
		case captureErr != nil:
			result.Failed++
			s.l.Warn("evaluation case not captured from a decided proposal",
				zap.String("proposalId", candidate.ProposalID.String()),
				zap.String("organizationId", candidate.OrganizationID.String()),
				zap.Error(captureErr),
			)
		case captured.Duplicate:
			result.Duplicates++
		default:
			result.Captured++
		}
	}

	return result, nil
}

func (s *Service) latestDecision(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	proposalID pulid.ID,
) (*agent.AgentDecision, error) {
	decisions, err := s.decisions.ListByProposals(
		ctx,
		repositories.ListAgentDecisionsByProposalsRequest{
			ProposalIDs: []pulid.ID{proposalID},
			TenantInfo:  tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, decision := range decisions {
		if decision.ProposalID != nil && *decision.ProposalID == proposalID {
			return decision, nil
		}
	}

	return nil, nil
}

func (s *Service) backgroundInput(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	run *agent.AgentRun,
) string {
	var subject *agentdefinition.RuntimeSubject
	if s.subjects != nil {
		described, err := s.subjects.Describe(ctx, tenantInfo, run.SubjectType, run.SubjectID)
		if err != nil {
			s.l.Warn("subject could not be described for an evaluation case",
				zap.String("runId", run.ID.String()),
				zap.Error(err),
			)
		} else {
			subject = described
		}
	}
	if subject == nil {
		subject = &agentdefinition.RuntimeSubject{
			Type:  run.SubjectType,
			ID:    run.SubjectID.String(),
			Label: string(run.SubjectType),
		}
	}

	return agentdefinition.BackgroundRunInput(run.Trigger, "", subject)
}

func (s *Service) freezeChat(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	anchor turnAnchor,
) (*frozenTurn, *redaction, error) {
	messages, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:     anchor.threadID,
		TenantInfo:   tenantInfo,
		ExcludeKinds: conversation.ModelHiddenKinds(),
	})
	if err != nil {
		return nil, nil, err
	}

	anchorIndex := -1
	if anchor.messageID.IsNotNil() {
		anchorIndex = slices.IndexFunc(messages, func(message conversation.Message) bool {
			return message.ID == anchor.messageID
		})
		if anchorIndex < 0 {
			return nil, nil, errortypes.NewNotFoundError(
				"The message is no longer in the conversation",
			)
		}
	}

	turn := -1
	for i := range messages {
		message := &messages[i]
		if message.Role != conversation.RoleUser {
			continue
		}
		if anchorIndex >= 0 {
			if i < anchorIndex {
				turn = i
			}
			continue
		}
		if anchor.before > 0 && message.CreatedAt > anchor.before {
			continue
		}
		turn = i
	}
	if turn < 0 {
		return nil, nil, errortypes.NewBusinessError(
			"The conversation no longer holds the question that led here",
		)
	}

	end := len(messages)
	for i := turn + 1; i < len(messages); i++ {
		if messages[i].Role == conversation.RoleUser {
			end = i
			break
		}
	}

	asked := &messages[turn]
	out := &redaction{}
	frozen := &frozenTurn{
		threadID: anchor.threadID,
		input:    asked.Content,
		history:  s.redactor.History(agentquality.HistoryFromMessages(messages[:turn]), out),
		page:     asked.PageContext,
		mentions: asked.Mentions,
	}
	if anchorIndex >= 0 {
		if messages[anchorIndex].Role == conversation.RoleAssistant {
			frozen.messageID = messages[anchorIndex].ID
		}
	} else {
		frozen.messageID = asked.ID
	}

	frozen.fixtures, frozen.called = s.fixtures(messages[turn+1:end], out)

	return frozen, out, nil
}

func (s *Service) fixtures(
	turn []conversation.Message,
	out *redaction,
) ([]agentquality.ToolFixture, []string) {
	args := make(map[string]map[string]any)
	called := make([]string, 0)
	fixtures := make([]agentquality.ToolFixture, 0)
	for i := range turn {
		message := &turn[i]
		switch message.Role {
		case conversation.RoleAssistant:
			for _, call := range message.ToolCalls {
				args[call.ID] = call.Arguments
				if !agentquality.IsRuntimeTool(call.Name) && !slices.Contains(called, call.Name) {
					called = append(called, call.Name)
				}
			}
		case conversation.RoleTool:
			if agentquality.IsRuntimeTool(message.ToolName) {
				continue
			}
			fixtures = append(fixtures, agentquality.ToolFixture{
				Tool: message.ToolName,
				Args: s.redactor.args(message.ToolName, args[message.ToolCallID], out),
				Result: s.redactor.decodedResult(
					message.ToolName,
					message.Content,
					message.ToolFailed,
					out,
				),
				Failed: message.ToolFailed,
			})
		}
	}

	return fixtures, called
}

func applyFrozen(
	evalCase *agentquality.EvalCase,
	frozen *frozenTurn,
	redacted *redaction,
	now int64,
) {
	threadID := frozen.threadID
	evalCase.SourceThreadID = &threadID
	if frozen.messageID.IsNotNil() {
		messageID := frozen.messageID
		evalCase.SourceMessageID = &messageID
	}
	evalCase.Input = frozen.input
	evalCase.History = frozen.history
	evalCase.PageContext = frozen.page
	evalCase.Mentions = frozen.mentions
	evalCase.ToolFixtures = frozen.fixtures
	evalCase.Redaction = &agentquality.Redaction{
		Fields:     redacted.fields,
		RedactedAt: now,
	}
	if evalCase.Redaction.Fields == nil {
		evalCase.Redaction.Fields = []agentquality.RedactedField{}
	}
}

func proposalTitle(toolName string, rejected bool) string {
	verdict := "Approved"
	if rejected {
		verdict = "Rejected"
	}

	return verdict + ": " + toolName
}

func titleFromInput(input string) string {
	collapsed := strings.Join(strings.Fields(input), " ")

	return stringutils.Ellipsize(collapsed, agentquality.MaxTitleChars-1)
}
