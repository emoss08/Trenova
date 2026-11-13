package agentevalcaseservice

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreplay"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationtranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
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
	askedAt   int64
	history   []agentquality.HistoryMessage
	page      *agentpagecontext.PageContext
	mentions  []agentpagecontext.EntityRef
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
	req *agentqualityports.CreateEvalCaseFromProposalRequest,
	actor *permissionports.RequestActor,
) (*agentqualityports.EvalCaseCapture, error) {
	return s.captureProposal(ctx, req, actor, false)
}

func (s *Service) captureProposal(
	ctx context.Context,
	req *agentqualityports.CreateEvalCaseFromProposalRequest,
	actor *permissionports.RequestActor,
	automatic bool,
) (*agentqualityports.EvalCaseCapture, error) {
	tenantInfo := req.TenantInfo
	existing, err := s.cases.GetByProposal(ctx, agentqualityports.GetAgentEvalCaseByProposalRequest{
		ProposalID: req.ProposalID,
		TenantInfo: tenantInfo,
	})
	switch {
	case err == nil:
		return &agentqualityports.EvalCaseCapture{Case: existing, Duplicate: true}, nil
	case !isNotFound(err):
		return nil, err
	}

	proposal, err := s.proposals.GetByID(ctx, agentproposalports.GetAgentProposalByIDRequest{
		ID:         req.ProposalID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	decided, err := s.proposalDecisions(ctx, tenantInfo, proposal.ID)
	if err != nil {
		return nil, err
	}
	if len(decided) == 0 {
		return nil, errortypes.NewBusinessError(
			"Only a proposal a person approved or rejected can become an evaluation case",
		)
	}

	run, err := s.runs.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
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

	definition, err := s.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         run.AgentDefinitionID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	evalCase := proposalCase(&proposalCapture{
		tenant:     tenantInfo,
		title:      req.Title,
		proposal:   proposal,
		decided:    decided,
		run:        run,
		definition: definition,
		automatic:  automatic,
	})

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
		evalCase.ReachedTools = beyondHeld(definition, proposal.ToolName, frozen.called)
		if len(evalCase.ReachedTools) > 0 {
			evalCase.Status = agentquality.CaseStatusCandidate
			evalCase.Automatic = false
		}
	} else {
		evalCase.Input = s.backgroundInput(ctx, tenantInfo, run)
		evalCase.SubjectType = run.SubjectType
		evalCase.SubjectID = run.SubjectID
		if freezeErr := s.freezeRun(ctx, tenantInfo, run, evalCase); freezeErr != nil {
			return nil, freezeErr
		}
	}

	return s.save(ctx, evalCase, actor)
}

type proposalCapture struct {
	tenant     pagination.TenantInfo
	title      string
	proposal   *agent.AgentProposal
	decided    []*agentdecision.AgentDecision
	run        *agent.AgentRun
	definition *agentdefinition.Definition
	automatic  bool
}

func proposalCase(p *proposalCapture) *agentquality.EvalCase {
	original := agentreplay.NewOriginalProposal(p.proposal, p.decided)
	rejected := original.Decision == agentdecision.DecisionRejected
	runID := p.run.ID
	proposalID := p.proposal.ID
	evalCase := &agentquality.EvalCase{
		OrganizationID:    p.tenant.OrgID,
		BusinessUnitID:    p.tenant.BuID,
		AgentDefinitionID: p.definition.ID,
		Title:             strings.TrimSpace(p.title),
		Source:            agentquality.CaseSourceDecidedProposal,
		Status:            agentquality.CaseStatusCandidate,
		Trigger:           p.run.Trigger,
		SourceRunID:       &runID,
		SourceProposalID:  &proposalID,
		HeldTools:         p.definition.EffectiveToolNames(),
		Expected: agentquality.Expected{
			ToolMode: agentquality.ToolMatchAnyOrder,
			Proposals: []agentquality.ExpectedProposal{{
				ToolName:         p.proposal.ToolName,
				Params:           original.Expected(),
				Rejected:         rejected,
				SourceProposalID: p.proposal.ID.String(),
			}},
		},
		CapturedFingerprint: agentquality.FingerprintOf(p.definition, promptVersion),
	}
	if evalCase.Title == "" {
		evalCase.Title = proposalTitle(p.proposal.ToolName, rejected)
	}
	if p.automatic {
		evalCase.Status = agentquality.CaseStatusActive
		evalCase.Automatic = true
	}

	return evalCase
}

func (s *Service) CreateFromMessage(
	ctx context.Context,
	req *agentqualityports.CreateEvalCaseFromMessageRequest,
	actor *permissionports.RequestActor,
) (*agentqualityports.EvalCaseCapture, error) {
	return s.captureMessage(ctx, &messageCapture{
		threadID:  req.ThreadID,
		messageID: req.MessageID,
		title:     req.Title,
		tenant:    req.TenantInfo,
	}, actor)
}

func (s *Service) CreateFromFeedback(
	ctx context.Context,
	req *agentqualityports.CreateEvalCaseFromFeedbackRequest,
	actor *permissionports.RequestActor,
) (*agentqualityports.EvalCaseCapture, error) {
	feedback, err := s.likedReply(ctx, req.TenantInfo, req.FeedbackID)
	if err != nil {
		return nil, err
	}

	captured, err := s.captureMessage(ctx, &messageCapture{
		threadID:  *feedback.ThreadID,
		messageID: feedback.TargetID,
		title:     req.Title,
		tenant:    req.TenantInfo,
		feedback:  feedback,
	}, actor)
	if err != nil {
		return nil, err
	}
	if feedback.EvalCaseID != nil && *feedback.EvalCaseID == captured.Case.ID {
		return captured, nil
	}

	if err = s.feedback.LinkEvalCase(ctx, aifeedbackports.LinkAIFeedbackEvalCaseRequest{
		TenantInfo: req.TenantInfo,
		FeedbackID: feedback.ID,
		EvalCaseID: captured.Case.ID,
	}); err != nil {
		return nil, err
	}

	return captured, nil
}

func (s *Service) likedReply(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	feedbackID pulid.ID,
) (*aifeedback.Feedback, error) {
	rows, err := s.feedback.ListByIDs(ctx, aifeedbackports.ListAIFeedbackByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        []pulid.ID{feedbackID},
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errortypes.NewNotFoundError("The rating is no longer on record")
	}

	feedback := rows[0]
	if feedback.Rating != aifeedback.RatingPositive {
		return nil, errortypes.NewBusinessError(
			"Only a reply someone rated as good can become an evaluation case",
		)
	}
	if feedback.TargetType != aifeedback.TargetAssistantMessage ||
		feedback.ThreadID == nil || feedback.ThreadID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"Only a rated reply in a conversation can become an evaluation case",
		)
	}

	return feedback, nil
}

type messageCapture struct {
	threadID  pulid.ID
	messageID pulid.ID
	title     string
	tenant    pagination.TenantInfo
	feedback  *aifeedback.Feedback
}

func (s *Service) captureMessage(
	ctx context.Context,
	req *messageCapture,
	actor *permissionports.RequestActor,
) (*agentqualityports.EvalCaseCapture, error) {
	tenantInfo := req.tenant
	thread, err := s.conversations.GetThreadOwned(ctx, conversationports.GetThreadOwnedRequest{
		ID:         req.threadID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	definition, err := s.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         thread.AgentDefinitionID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	frozen, redacted, err := s.freezeChat(ctx, tenantInfo, turnAnchor{
		threadID:  thread.ID,
		messageID: req.messageID,
	})
	if err != nil {
		return nil, err
	}
	if frozen.messageID != req.messageID {
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
		Title:             strings.TrimSpace(req.title),
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
	if req.feedback != nil {
		feedbackID := req.feedback.ID
		evalCase.SourceFeedbackID = &feedbackID
		evalCase.SourceTurnID = req.feedback.TurnID
		evalCase.SourceRunID = req.feedback.RunID
	}
	applyFrozen(evalCase, frozen, redacted, s.now())
	evalCase.ReachedTools = beyondHeld(definition, "", frozen.called)
	if evalCase.Title == "" {
		evalCase.Title = titleFromInput(evalCase.Input)
	}

	return s.save(ctx, evalCase, actor)
}

type agentScope struct {
	tenant  pagination.TenantInfo
	agentID pulid.ID
}

func (s *Service) CaptureCandidates(
	ctx context.Context,
	req agentqualityports.CaptureEvalCaseCandidatesRequest,
) (*agentqualityports.CaptureEvalCaseCandidatesResult, error) {
	candidates, err := s.cases.ListCaptureCandidates(
		ctx,
		agentqualityports.ListEvalCaseCaptureCandidatesRequest(req),
	)
	if err != nil {
		return nil, err
	}

	result := &agentqualityports.CaptureEvalCaseCandidatesResult{Scanned: len(candidates)}
	controls := make(map[pagination.TenantInfo]*agentqualitycontrol.Control)
	grown := make(map[agentScope]int)
	for _, candidate := range candidates {
		tenantInfo := candidate.TenantInfo()
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		control := s.controlOf(tenantCtx, tenantInfo, controls)
		captured, captureErr := s.captureProposal(
			tenantCtx,
			&agentqualityports.CreateEvalCaseFromProposalRequest{
				ProposalID: candidate.ProposalID,
				TenantInfo: tenantInfo,
			},
			nil,
			control.AutoActivate,
		)
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
			if captured.Case.Automatic {
				result.Activated++
				grown[agentScope{
					tenant:  tenantInfo,
					agentID: captured.Case.AgentDefinitionID,
				}] = control.AutoCasesPerAgent
			}
		}
	}

	result.Adopted = s.adopt(ctx, req.Limit, controls, grown)

	for scope, keep := range grown {
		result.Rotated += s.rotate(
			dbscope.WithTenant(ctx, scope.tenant.DBTenant()),
			scope,
			keep,
		)
	}

	return result, nil
}

const maxAdoptBatches = 5

func (s *Service) adopt(
	ctx context.Context,
	limit int,
	controls map[pagination.TenantInfo]*agentqualitycontrol.Control,
	grown map[agentScope]int,
) int {
	now := s.now()
	adopted := 0
	for range maxAdoptBatches {
		batch, err := s.cases.AdoptCandidates(ctx, agentqualityports.AdoptEvalCaseCandidatesRequest{
			CreatedAfter: agentqualitycontrol.AutomaticCaseCutoff(now),
			At:           now,
			Limit:        limit,
		})
		if err != nil {
			s.l.Warn("candidate evaluation cases not adopted this round", zap.Error(err))

			return adopted
		}
		for i := range batch {
			tenantInfo := batch[i].TenantInfo()
			control := s.controlOf(
				dbscope.WithTenant(ctx, tenantInfo.DBTenant()),
				tenantInfo,
				controls,
			)
			grown[agentScope{
				tenant:  tenantInfo,
				agentID: batch[i].AgentDefinitionID,
			}] = control.AutoCasesPerAgent
		}
		adopted += len(batch)
		if len(batch) == 0 || (limit > 0 && len(batch) < limit) {
			return adopted
		}
	}

	return adopted
}

func (s *Service) controlOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	read map[pagination.TenantInfo]*agentqualitycontrol.Control,
) *agentqualitycontrol.Control {
	if control, ok := read[tenantInfo]; ok {
		return control
	}

	control, err := s.settings.Control(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("quality controls unread; decided proposals stay candidates this round",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err),
		)
		control = &agentqualitycontrol.Control{}
	}
	read[tenantInfo] = control

	return control
}

func (s *Service) rotate(ctx context.Context, scope agentScope, keep int) int {
	if keep <= 0 {
		return 0
	}

	cases, err := s.cases.ListAutomatic(ctx, agentqualityports.ListAutomaticEvalCasesRequest{
		TenantInfo:        scope.tenant,
		AgentDefinitionID: scope.agentID,
	})
	if err != nil {
		s.l.Warn("automatic evaluation cases unread; the agent keeps more than its share",
			zap.String("agentDefinitionId", scope.agentID.String()),
			zap.Error(err),
		)

		return 0
	}

	overflow := agentquality.AutomaticOverflow(cases, keep)
	if len(overflow) == 0 {
		return 0
	}

	retired, err := s.cases.RetireAutomatic(ctx, &agentqualityports.RetireAutomaticEvalCasesRequest{
		TenantInfo: scope.tenant,
		IDs:        overflow,
		At:         s.now(),
	})
	if err != nil {
		s.l.Warn("automatic evaluation cases not retired; the agent keeps more than its share",
			zap.String("agentDefinitionId", scope.agentID.String()),
			zap.Error(err),
		)

		return 0
	}
	if retired > 0 {
		s.logRotation(ctx, scope, overflow, keep)
	}

	return retired
}

func (s *Service) proposalDecisions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	proposalID pulid.ID,
) ([]*agentdecision.AgentDecision, error) {
	decisions, err := s.decisions.ListByProposals(
		ctx,
		agentdecisionports.ListAgentDecisionsByProposalsRequest{
			ProposalIDs: []pulid.ID{proposalID},
			TenantInfo:  tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	return agentreplay.DecisionsByProposal(decisions)[proposalID], nil
}

func (s *Service) backgroundInput(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	run *agent.AgentRun,
) string {
	var subject *agentturn.RuntimeSubject
	described, err := s.subjects.Describe(ctx, tenantInfo, run.SubjectType, run.SubjectID)
	if err != nil {
		s.l.Warn("subject could not be described for an evaluation case",
			zap.String("runId", run.ID.String()),
			zap.Error(err),
		)
	} else {
		subject = described
	}
	if subject == nil {
		subject = &agentturn.RuntimeSubject{
			Type:  run.SubjectType,
			ID:    run.SubjectID.String(),
			Label: string(run.SubjectType),
		}
	}

	return agentmessage.BackgroundRunInput(run.Trigger, "", subject)
}

func (s *Service) freezeChat(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	anchor turnAnchor,
) (*frozenTurn, *redaction, error) {
	messages, err := s.conversations.ListMessages(ctx, &conversationports.ListMessagesRequest{
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
		askedAt:  asked.CreatedAt,
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

func beyondHeld(
	definition *agentdefinition.Definition,
	proposed string,
	called []string,
) []string {
	held := definition.EffectiveToolNames()
	reached := make([]string, 0)
	for _, name := range append([]string{proposed}, called...) {
		if name == "" || agentquality.IsRuntimeTool(name) || slices.Contains(held, name) ||
			slices.Contains(reached, name) {
			continue
		}
		reached = append(reached, name)
	}

	return reached
}

func (s *Service) freezeRun(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	run *agent.AgentRun,
	evalCase *agentquality.EvalCase,
) error {
	rows, err := s.runs.ListTranscriptsByIDs(ctx, &agentruntimeports.ListAgentRunsByIDsRequest{
		IDs:        []pulid.ID{run.ID},
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if len(rows) == 0 || rows[0].Transcript == nil {
		return nil
	}

	entries := rows[0].Transcript.Messages
	messages := make([]conversation.Message, 0, len(entries))
	for i := range entries {
		if entries[i].Omitted {
			continue
		}
		messages = append(messages, conversationtranscript.MessageOfTranscript(&entries[i]))
	}

	out := &redaction{}
	evalCase.ToolFixtures, _ = s.fixtures(messages, out)
	evalCase.Redaction = &agentquality.Redaction{
		Fields:     out.fields,
		RedactedAt: s.now(),
	}
	if evalCase.Redaction.Fields == nil {
		evalCase.Redaction.Fields = []agentquality.RedactedField{}
	}
	if started := cmp.Or(run.StartedAt, run.CreatedAt); started > 0 {
		evalCase.AskedAt = &started
	}

	return nil
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
			if _, known := args[message.ToolCallID]; !known ||
				len(fixtures) >= agentquality.MaxToolFixtures {
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
	if frozen.askedAt > 0 {
		askedAt := frozen.askedAt
		evalCase.AskedAt = &askedAt
	}
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
