package aiauditservice

import (
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	executionWindowSeconds = 60
	payloadKeyFailed       = "failed"
	payloadKeyName         = "name"
	payloadKeyContent      = "content"
	payloadKeyAt           = "at"
	payloadKeyAgentID      = "agentId"
	payloadKeyDelegateCall = "delegateCallId"
	payloadKeyStatus       = "status"
	payloadKeyReason       = "reason"
	payloadKeyReply        = "reply"
)

// deriver turns source rows into trail rows. It reads nothing itself: what
// it needs about owners and names arrives in lookups, so a row derives the
// same whenever it is read.
type deriver struct {
	redactor *Redactor
}

func sourceKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func (d *deriver) base(o *owner) *aiaudit.AIAuditEvent {
	event := &aiaudit.AIAuditEvent{
		PrincipalType:          o.principalType,
		PrincipalID:            o.principalID,
		OnBehalfOfUserID:       o.onBehalfOf,
		AgentDefinitionID:      o.agentID,
		AgentDefinitionVersion: o.agentVersion,
		OwnerID:                o.id,
		RunID:                  o.runID,
		TurnID:                 o.turnID,
		ThreadID:               o.threadID,
		TraceID:                o.traceID,
		DelegateCallID:         o.delegateCallID,
		ParentOwnerID:          o.parentOwnerID,
		Purpose:                o.purpose,
		Simulated:              o.simulated,
	}
	if o.id.IsNotNil() {
		event.OwnerKind = string(o.kind)
	}
	if event.PrincipalType == "" {
		event.PrincipalType = aiaudit.PrincipalSystem
	}
	if event.Purpose == "" {
		event.Purpose = aiaudit.PurposeLive
	}

	return event
}

func withTaint(event *aiaudit.AIAuditEvent, tainted bool, taint *agent.RunTaint) {
	event.Tainted = tainted || taint.Tainted()
	if taint.Tainted() {
		event.Taint = taint
		event.ExternalContent = slices.ContainsFunc(taint.Marks, func(mark agent.TaintMark) bool {
			return mark.Source == agent.TaintSourceWeb
		})
	}
}

func (d *deriver) runEvents(l *lookups, run *agent.AgentRun) []*aiaudit.AIAuditEvent {
	o := l.ownerOf(string(agent.RunOwnerAgentRun), run.ID)
	events := make([]*aiaudit.AIAuditEvent, 0, 2)

	started := d.base(&o)
	started.SourceKey = sourceKey("run", run.ID.String(), "started")
	started.Kind = aiaudit.KindRunStarted
	started.Outcome = aiaudit.OutcomeStarted
	started.OccurredAt = firstPositive(run.StartedAt, run.CreatedAt)
	started.EntityType = string(run.SubjectType)
	started.EntityID = run.SubjectID.String()
	started.ResultSummary = string(run.Trigger)
	started.Simulated = false
	events = append(events, started)

	if run.CompletedAt == nil {
		return events
	}

	ended := d.base(&o)
	ended.SourceKey = sourceKey("run", run.ID.String(), "ended")
	ended.Kind = aiaudit.KindRunEnded
	ended.Outcome = runOutcome(run.Status)
	ended.OccurredAt = *run.CompletedAt
	ended.EntityType = string(run.SubjectType)
	ended.EntityID = run.SubjectID.String()
	ended.Reason = d.redactor.Text(run.ErrorMessage, aiaudit.MaxReasonLength)
	ended.ResultSummary = d.redactor.Text(run.Summary, aiaudit.MaxResultSummaryLength)
	ended.WindowStart = started.OccurredAt
	ended.WindowEnd = ended.OccurredAt
	withTaint(ended, run.Tainted, run.Taint)
	events = append(events, ended)

	return events
}

func runOutcome(status agent.RunStatus) aiaudit.Outcome {
	switch status {
	case agent.RunStatusFailed:
		return aiaudit.OutcomeFailed
	case agent.RunStatusCompleted, agent.RunStatusShadowCompleted, agent.RunStatusAwaitingDecision:
		return aiaudit.OutcomeCompleted
	case agent.RunStatusPending, agent.RunStatusGatheringContext, agent.RunStatusDiagnosing:
		return aiaudit.OutcomeStopped
	default:
		return aiaudit.OutcomeCompleted
	}
}

func (d *deriver) turnEvents(
	l *lookups,
	turn *conversation.AssistantTurn,
) []*aiaudit.AIAuditEvent {
	o := l.ownerOf(string(agent.RunOwnerAssistantTurn), turn.ID)
	events := make([]*aiaudit.AIAuditEvent, 0, 2)

	started := d.base(&o)
	started.SourceKey = sourceKey("turn", turn.ID.String(), "started")
	started.Kind = aiaudit.KindRunStarted
	started.Outcome = aiaudit.OutcomeStarted
	started.OccurredAt = firstPositive(turn.StartedAt, turn.CreatedAt)
	started.ResultSummary = string(turn.Origin)
	started.Reconstructed = o.agentID.IsNil()
	events = append(events, started)

	if !turn.Status.Terminal() {
		return events
	}

	ended := d.base(&o)
	ended.SourceKey = sourceKey("turn", turn.ID.String(), "ended")
	ended.Kind = aiaudit.KindRunEnded
	ended.Outcome = turnOutcome(turn.Status)
	ended.OccurredAt = turn.UpdatedAt
	if turn.CompletedAt != nil {
		ended.OccurredAt = *turn.CompletedAt
	}
	ended.Reason = d.redactor.Text(turn.ErrorMessage, aiaudit.MaxReasonLength)
	ended.WindowStart = started.OccurredAt
	ended.WindowEnd = ended.OccurredAt
	ended.Reconstructed = started.Reconstructed
	events = append(events, ended)

	return events
}

func turnOutcome(status conversation.AssistantTurnStatus) aiaudit.Outcome {
	switch status {
	case conversation.AssistantTurnStatusRefused:
		return aiaudit.OutcomeRefused
	case conversation.AssistantTurnStatusStopped:
		return aiaudit.OutcomeStopped
	case conversation.AssistantTurnStatusFailed:
		return aiaudit.OutcomeFailed
	case conversation.AssistantTurnStatusCompleted,
		conversation.AssistantTurnStatusPending,
		conversation.AssistantTurnStatusRunning:
		return aiaudit.OutcomeCompleted
	default:
		return aiaudit.OutcomeCompleted
	}
}

func (d *deriver) usageEvent(l *lookups, record *aiusage.AIUsageRecord) *aiaudit.AIAuditEvent {
	o, reconstructed := d.usageOwner(l, record)

	event := d.base(&o)
	event.SourceKey = sourceKey("usage", record.ID.String())
	event.Kind = aiaudit.KindModelCall
	event.Outcome = aiaudit.OutcomeSucceeded
	if !record.Succeeded {
		event.Outcome = aiaudit.OutcomeFailed
		event.Reason = d.redactor.Text(
			joinReason(record.ErrorClass, record.ErrorMessage),
			aiaudit.MaxReasonLength,
		)
	}
	event.OccurredAt = record.CreatedAt
	event.Reconstructed = reconstructed

	applyUsageAttribution(event, record, o.id)
	applyUsageCall(event, record)

	return event
}

func applyUsageAttribution(
	event *aiaudit.AIAuditEvent,
	record *aiusage.AIUsageRecord,
	ownerID pulid.ID,
) {
	if record.UserID.IsNotNil() {
		event.OnBehalfOfUserID = record.UserID
	}
	if record.AgentDefinitionID.IsNotNil() {
		event.AgentDefinitionID = record.AgentDefinitionID
	}
	if record.AgentDefinitionVersion != nil {
		event.AgentDefinitionVersion = record.AgentDefinitionVersion
	}
	if record.RunID.IsNotNil() {
		event.RunID = record.RunID
	}
	if record.ThreadID.IsNotNil() {
		event.ThreadID = record.ThreadID
	}
	if record.DelegateCallID != "" {
		event.DelegateCallID = record.DelegateCallID
	}
	if record.TraceID != "" {
		event.TraceID = record.TraceID
	}
	event.SpanID = record.SpanID
	if ownerID.IsNil() {
		event.PrincipalType, event.PrincipalID = usagePrincipal(record)
	}
	if record.Surface == aiusage.SurfaceEvaluation {
		event.Purpose = aiaudit.PurposeEvaluation
	}
}

func applyUsageCall(event *aiaudit.AIAuditEvent, record *aiusage.AIUsageRecord) {
	event.ProviderID = record.ProviderID
	event.ProviderKind = string(record.ProviderKind)
	event.Model = record.Model
	if record.Attempt > 0 {
		attempt := record.Attempt
		event.Attempt = &attempt
	}
	event.Failover = record.Failover
	event.InputTokens = record.InputTokens
	event.OutputTokens = record.OutputTokens
	event.ReasoningTokens = record.ReasoningTokens
	event.CacheReadTokens = record.CacheReadTokens
	event.CacheWriteTokens = record.CacheWriteTokens
	event.CostUSD = record.CostUSD
	latency := record.LatencyMs
	event.LatencyMs = &latency
	event.EntityType = string(record.SubjectType)
	event.EntityID = record.SubjectID
	event.ResultSummary = string(record.Surface)
	if record.Feature != "" {
		event.ResultSummary = string(record.Feature)
	}
	event.WindowStart = record.CreatedAt - (record.LatencyMs+999)/1000
	event.WindowEnd = record.CreatedAt
}

// usageOwner is the run or turn a usage row was for: named on the row once
// the runtime writes it, else worked out from the run it names, or from the
// turn of its conversation that was running when it was written.
func (d *deriver) usageOwner(l *lookups, record *aiusage.AIUsageRecord) (owner, bool) {
	if record.OwnerID.IsNotNil() {
		return l.ownerOf(string(record.OwnerKind), record.OwnerID), false
	}
	if record.RunID.IsNotNil() {
		return l.ownerOf(string(agent.RunOwnerAgentRun), record.RunID), true
	}
	if record.ThreadID.IsNotNil() {
		if turn := l.turnAt(record.ThreadID, record.CreatedAt); turn != nil {
			return l.ownerOf(string(agent.RunOwnerAssistantTurn), turn.ID), true
		}
	}

	return owner{purpose: aiaudit.PurposeLive}, true
}

func usagePrincipal(record *aiusage.AIUsageRecord) (aiaudit.PrincipalType, string) {
	switch {
	case record.UserID.IsNotNil():
		return aiaudit.PrincipalUser, record.UserID.String()
	case record.AgentDefinitionID.IsNotNil():
		return aiaudit.PrincipalAgent, record.AgentDefinitionID.String()
	default:
		return aiaudit.PrincipalSystem, serviceports.SystemPrincipalID.String()
	}
}

func joinReason(class, message string) string {
	switch {
	case class == "":
		return message
	case message == "":
		return class
	default:
		return class + ": " + message
	}
}

func decodeOutcome(raw map[string]any) serviceports.RunStepOutcome {
	var outcome serviceports.RunStepOutcome
	if len(raw) == 0 {
		return outcome
	}
	encoded, err := sonic.Marshal(raw)
	if err != nil {
		return outcome
	}
	_ = sonic.Unmarshal(encoded, &outcome)

	return outcome
}

// stepEvent is a settled tool call: what it was asked, how it was decided,
// what it did.
func (d *deriver) stepEvent(l *lookups, step *agent.AgentRunStep) (*aiaudit.AIAuditEvent, error) {
	outcome := decodeOutcome(step.Outcome)
	event, err := d.stepBase(l, step)
	if err != nil {
		return nil, err
	}

	event.Outcome = stepOutcome(step, &outcome)
	event.Reason = d.redactor.Text(outcome.Reason, aiaudit.MaxReasonLength)
	event.WindowEnd = step.UpdatedAt
	if len(outcome.Taint) > 0 {
		taint := &agent.RunTaint{Marks: outcome.Taint}
		withTaint(event, false, taint)
	}
	if outcome.Action != nil {
		d.applyAction(event, outcome.Action)
	}

	return event, nil
}

// unknownStepEvent is a call that was begun and never settled, in a run or
// turn that has ended: whether it ran cannot be known.
func (d *deriver) unknownStepEvent(
	l *lookups,
	step *agent.AgentRunStep,
) (*aiaudit.AIAuditEvent, error) {
	event, err := d.stepBase(l, step)
	if err != nil {
		return nil, err
	}
	event.Outcome = aiaudit.OutcomeUnknown
	event.WindowEnd = step.UpdatedAt

	return event, nil
}

func (d *deriver) stepBase(l *lookups, step *agent.AgentRunStep) (*aiaudit.AIAuditEvent, error) {
	o := l.ownerOf(step.OwnerKind, step.OwnerID)

	event := d.base(&o)
	event.SourceKey = sourceKey("step", step.OwnerID.String(), step.StepKey)
	event.Kind = aiaudit.KindToolCall
	event.OccurredAt = step.CreatedAt
	event.WindowStart = step.CreatedAt
	event.StepKey = step.StepKey
	event.CallID = step.CallID
	event.ToolName = step.ToolName
	event.Reconstructed = step.AgentDefinitionID.IsNil() || step.TraceID == ""

	if step.AgentDefinitionID.IsNotNil() {
		event.AgentDefinitionID = step.AgentDefinitionID
	}
	if step.AgentDefinitionVersion != nil {
		event.AgentDefinitionVersion = step.AgentDefinitionVersion
	}
	if step.DelegateCallID != "" {
		event.DelegateCallID = step.DelegateCallID
		event.ParentOwnerID = step.OwnerID
	}
	if step.TraceID != "" {
		event.TraceID = step.TraceID
	}
	event.SpanID = step.SpanID

	if policy, ok := d.redactor.Policy(step.ToolName); ok {
		event.ToolEffect = string(policy.EffectiveEffect())
		if len(policy.Egress) == 1 {
			event.EgressClass = string(policy.Egress[0])
		}
	}

	if err := d.recordArguments(event, step.ToolName, step.Arguments); err != nil {
		return nil, err
	}

	return event, nil
}

func stepOutcome(step *agent.AgentRunStep, outcome *serviceports.RunStepOutcome) aiaudit.Outcome {
	if action := outcome.Action; action != nil {
		switch {
		case action.Simulated:
			return aiaudit.OutcomeSimulated
		case action.Executed && action.ExecutionError != "":
			return aiaudit.OutcomeFailed
		case action.Executed:
			return aiaudit.OutcomeRan
		default:
			return aiaudit.OutcomeProposed
		}
	}

	failed := outcome.Failed || step.Status == string(serviceports.RunStepFailed)
	switch {
	case failed && outcome.Reason != "":
		return aiaudit.OutcomeDenied
	case failed:
		return aiaudit.OutcomeFailed
	default:
		return aiaudit.OutcomeRan
	}
}

func (d *deriver) applyAction(event *aiaudit.AIAuditEvent, action *serviceports.PendingAction) {
	event.Tier = string(action.Tier)
	event.TierSource = string(action.TierSource)
	event.HeldBy = slices.Clone(action.HeldBy)
	if action.Egress != "" {
		event.EgressClass = string(action.Egress)
	}
	if action.Tainted {
		event.Tainted = true
	}
	event.Simulated = event.Simulated || action.Simulated
	event.ProposalID = action.ProposalID
	if action.TraceID != "" && event.TraceID == "" {
		event.TraceID = action.TraceID
	}
	if action.SpanID != "" && event.SpanID == "" {
		event.SpanID = action.SpanID
	}
	if target := action.Target; target != nil {
		event.EntityType = target.Resource.String()
		event.EntityID = target.ID.String()
		version := target.Version
		event.VersionBefore = &version
	}
	if record := resultRecord(action.ExecutionResult); record != nil {
		if event.EntityID == "" {
			event.EntityType = record.EntityType
			event.EntityID = record.ID
		}
	}
	if action.ExecutedVersion != nil {
		after := *action.ExecutedVersion
		event.VersionAfter = &after
	}
	event.ResultSummary = d.redactor.Text(
		executionSummary(action.ExecutionResult, action.ExecutionError),
		aiaudit.MaxResultSummaryLength,
	)
}

func resultRecord(result *agent.ToolExecutionResult) *agent.RecordRef {
	if result == nil {
		return nil
	}

	return result.Record.Bounded()
}

func executionSummary(result *agent.ToolExecutionResult, executionError string) string {
	if executionError != "" {
		return executionError
	}
	if result == nil {
		return ""
	}

	parts := make([]string, 0, 3)
	for _, part := range []string{result.Action, result.Kind, result.Name} {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, " ")
}

func (d *deriver) recordArguments(
	event *aiaudit.AIAuditEvent,
	toolName string,
	args map[string]any,
) error {
	redacted, err := d.redactor.Arguments(toolName, args)
	if err != nil {
		return err
	}
	if len(redacted.Values) > 0 {
		event.Arguments = redacted.Values
	}
	event.ArgumentSensitivity = redacted.Sensitivity
	event.RedactedPaths = redacted.RedactedPaths
	event.ArgumentsTruncated = redacted.Truncated

	return nil
}

// runEventEvent reads the edges of a delegated task, and a tool call that
// finished failed with no step behind it: refused before anything was
// claimed, so the step ledger never saw it.
func (d *deriver) runEventEvent(l *lookups, row *agent.AgentRunEvent) *aiaudit.AIAuditEvent {
	o := l.ownerOf(row.OwnerKind, row.OwnerID)
	payload := row.Payload
	at := int64Of(payload[payloadKeyAt])
	if at <= 0 {
		at = row.OccurredAt
	}

	event := d.base(&o)
	event.SourceKey = sourceKey("event", row.ID.String())
	event.OccurredAt = at
	event.CallID = firstNonEmpty(row.CallID, stringOf(payload[delegateOrCallKey(row.Kind)]))
	event.StepKey = row.StepKey
	event.Reconstructed = at == row.OccurredAt
	delegateCall := stringOf(payload[payloadKeyDelegateCall])
	if delegateCall != "" {
		event.DelegateCallID = delegateCall
		event.ParentOwnerID = row.OwnerID
	}
	if agentID := pulid.ID(stringOf(payload[payloadKeyAgentID])); pulid.LooksLike(
		agentID.String(),
	) {
		event.AgentDefinitionID = agentID
		event.AgentDefinitionVersion = nil
	}

	switch row.Kind {
	case serviceports.AssistantEventToolFinished:
		if failed, _ := payload[payloadKeyFailed].(bool); !failed {
			return nil
		}
		if _, claimed := l.calls[repositories.OwnerCall{
			OwnerID: row.OwnerID,
			CallID:  row.CallID,
		}]; claimed {
			return nil
		}
		event.Kind = aiaudit.KindToolRefused
		event.Outcome = aiaudit.OutcomeRefused
		event.ToolName = stringOf(payload[payloadKeyName])
		event.Reason = d.redactor.Text(
			stringOf(payload[payloadKeyContent]),
			aiaudit.MaxReasonLength,
		)
		if policy, ok := d.redactor.Policy(event.ToolName); ok {
			event.ToolEffect = string(policy.EffectiveEffect())
		}
	case serviceports.AssistantEventDelegateStarted:
		event.Kind = aiaudit.KindDelegationStarted
		event.Outcome = aiaudit.OutcomeStarted
		event.ToolName = delegateToolName
	case serviceports.AssistantEventDelegateFinished:
		event.Kind = aiaudit.KindDelegationEnded
		event.Outcome = delegateOutcome(
			conversation.DelegateStatus(stringOf(payload[payloadKeyStatus])),
		)
		event.ToolName = delegateToolName
		event.Reason = d.redactor.Text(stringOf(payload[payloadKeyReason]), aiaudit.MaxReasonLength)
		event.ResultSummary = d.redactor.Text(
			stringOf(payload[payloadKeyReply]),
			aiaudit.MaxResultSummaryLength,
		)
	default:
		return nil
	}
	event.WindowStart = at
	event.WindowEnd = at

	return event
}

const delegateToolName = "delegate_task"

func delegateOrCallKey(kind string) string {
	if kind == serviceports.AssistantEventToolFinished {
		return "callId"
	}

	return payloadKeyDelegateCall
}

func delegateOutcome(status conversation.DelegateStatus) aiaudit.Outcome {
	switch status {
	case conversation.DelegateStatusCompleted:
		return aiaudit.OutcomeCompleted
	case conversation.DelegateStatusExhausted:
		return aiaudit.OutcomeExhausted
	case conversation.DelegateStatusRefused:
		return aiaudit.OutcomeRefused
	case conversation.DelegateStatusDeclined:
		return aiaudit.OutcomeDeclined
	case conversation.DelegateStatusStopped:
		return aiaudit.OutcomeStopped
	case conversation.DelegateStatusFailed:
		return aiaudit.OutcomeFailed
	default:
		return aiaudit.OutcomeFailed
	}
}

func (d *deriver) proposalEvents(
	l *lookups,
	proposal *agent.AgentProposal,
) ([]*aiaudit.AIAuditEvent, error) {
	o := l.ownerOf(string(agent.RunOwnerAgentRun), proposal.RunID)
	events := make([]*aiaudit.AIAuditEvent, 0, 2)

	filed, err := d.proposalBase(&o, proposal)
	if err != nil {
		return nil, err
	}
	filed.SourceKey = sourceKey("proposal", proposal.ID.String(), "filed")
	filed.Kind = aiaudit.KindProposalFiled
	filed.Outcome = aiaudit.OutcomeFiled
	filed.OccurredAt = proposal.CreatedAt
	filed.WindowStart = proposal.CreatedAt
	filed.WindowEnd = proposal.CreatedAt
	filed.ResultSummary = d.redactor.Text(proposal.Rationale, aiaudit.MaxResultSummaryLength)
	filed.Simulated = false
	events = append(events, filed)

	if executed, ok := d.proposalExecution(&o, proposal); ok {
		executed.Arguments = filed.Arguments
		executed.ArgumentSensitivity = filed.ArgumentSensitivity
		executed.RedactedPaths = filed.RedactedPaths
		executed.ArgumentsTruncated = filed.ArgumentsTruncated
		events = append(events, executed)
	}

	if proposal.Status == agent.ProposalStatusExpired {
		expired, baseErr := d.proposalBase(&o, proposal)
		if baseErr != nil {
			return nil, baseErr
		}
		expired.SourceKey = sourceKey("proposal", proposal.ID.String(), "expired")
		expired.Kind = aiaudit.KindProposalExpired
		expired.Outcome = aiaudit.OutcomeExpired
		expired.OccurredAt = proposal.UpdatedAt
		if proposal.ExpiresAt > 0 && proposal.ExpiresAt <= proposal.UpdatedAt {
			expired.OccurredAt = proposal.ExpiresAt
		}
		expired.PrincipalType = aiaudit.PrincipalSystem
		expired.PrincipalID = serviceports.SystemPrincipalID.String()
		expired.Arguments = nil
		expired.ArgumentSensitivity = nil
		expired.RedactedPaths = nil
		expired.ArgumentsTruncated = false
		events = append(events, expired)
	}

	return events, nil
}

func (d *deriver) proposalBase(
	o *owner,
	proposal *agent.AgentProposal,
) (*aiaudit.AIAuditEvent, error) {
	event := d.base(o)
	event.ProposalID = proposal.ID
	if proposal.PlanID != nil {
		event.PlanID = *proposal.PlanID
	}
	event.ToolName = proposal.ToolName
	event.StepKey = proposal.StepKey
	event.Tier = string(proposal.AutonomyTier)
	event.HeldBy = slices.Clone(proposal.HeldBy)
	event.EgressClass = string(proposal.EgressClass)
	if proposal.TraceID != "" {
		event.TraceID = proposal.TraceID
	}
	event.SpanID = proposal.SpanID
	event.Reconstructed = proposal.StepKey == "" || proposal.TraceID == ""
	withTaint(event, proposal.Tainted, proposal.Taint)
	if proposal.TargetResource != "" {
		event.EntityType = proposal.TargetResource
		event.EntityID = proposal.TargetID.String()
		if proposal.TargetVersion > 0 {
			version := proposal.TargetVersion
			event.VersionBefore = &version
		}
	}
	if policy, ok := d.redactor.Policy(proposal.ToolName); ok {
		event.ToolEffect = string(policy.EffectiveEffect())
	}
	if err := d.recordArguments(event, proposal.ToolName, proposal.ToolParams); err != nil {
		return nil, err
	}

	return event, nil
}

// proposalExecution is what became of a proposal's write, once there is
// something to say: it ran, it failed, or it was previewed in simulation.
func (d *deriver) proposalExecution(
	o *owner,
	proposal *agent.AgentProposal,
) (*aiaudit.AIAuditEvent, bool) {
	var kind aiaudit.Kind
	var outcome aiaudit.Outcome
	var at int64

	switch {
	case proposal.SimulatedAt != nil:
		kind, outcome, at = aiaudit.KindProposalSimulated, aiaudit.OutcomeSimulated, *proposal.SimulatedAt
	case proposal.ExecutionError != "" || proposal.Status == agent.ProposalStatusExecutionFailed:
		kind, outcome = aiaudit.KindProposalExecutionFailed, aiaudit.OutcomeFailed
		at = proposal.UpdatedAt
		if proposal.ExecutedAt != nil {
			at = *proposal.ExecutedAt
		}
	case proposal.ExecutedAt != nil || proposal.Status == agent.ProposalStatusExecuted:
		kind, outcome = aiaudit.KindProposalExecuted, aiaudit.OutcomeSucceeded
		at = proposal.UpdatedAt
		if proposal.ExecutedAt != nil {
			at = *proposal.ExecutedAt
		}
	default:
		return nil, false
	}

	event := d.base(o)
	event.SourceKey = sourceKey("proposal", proposal.ID.String(), "executed")
	event.Kind = kind
	event.Outcome = outcome
	event.OccurredAt = at
	event.ProposalID = proposal.ID
	if proposal.PlanID != nil {
		event.PlanID = *proposal.PlanID
	}
	event.ToolName = proposal.ToolName
	event.StepKey = proposal.StepKey
	event.Tier = string(proposal.AutonomyTier)
	event.HeldBy = slices.Clone(proposal.HeldBy)
	event.EgressClass = string(proposal.EgressClass)
	event.TraceID = firstNonEmpty(proposal.TraceID, event.TraceID)
	event.SpanID = proposal.SpanID
	event.Simulated = kind == aiaudit.KindProposalSimulated
	withTaint(event, proposal.Tainted, proposal.Taint)
	if policy, ok := d.redactor.Policy(proposal.ToolName); ok {
		event.ToolEffect = string(policy.EffectiveEffect())
	}

	switch {
	case proposal.ExecutedByUserID.IsNotNil():
		event.PrincipalType = aiaudit.PrincipalUser
		event.PrincipalID = proposal.ExecutedByUserID.String()
	case proposal.AutonomyTier == agent.TierAutoExecute:
		event.PrincipalType = aiaudit.PrincipalAgent
		event.PrincipalID = o.agentID.String()
	default:
		event.Reconstructed = true
	}

	if proposal.TargetResource != "" {
		event.EntityType = proposal.TargetResource
		event.EntityID = proposal.TargetID.String()
		if proposal.TargetVersion > 0 {
			version := proposal.TargetVersion
			event.VersionBefore = &version
		}
	}
	if record := resultRecord(proposal.ExecutionResult); record != nil && event.EntityID == "" {
		event.EntityType = record.EntityType
		event.EntityID = record.ID
	}
	if proposal.ExecutedTargetVersion != nil {
		after := *proposal.ExecutedTargetVersion
		event.VersionAfter = &after
	}
	event.Reason = d.redactor.Text(proposal.ExecutionError, aiaudit.MaxReasonLength)
	event.ResultSummary = d.redactor.Text(
		executionSummary(proposal.ExecutionResult, ""),
		aiaudit.MaxResultSummaryLength,
	)

	event.WindowEnd = at
	event.WindowStart = at - executionWindowSeconds
	if proposal.ExecutedByUserID.IsNil() && proposal.AutonomyTier == agent.TierAutoExecute {
		event.WindowStart = proposal.CreatedAt - executionWindowSeconds
		if run, ok := o.startedAt(); ok {
			event.WindowStart = run
		}
	}

	return event, true
}

func (d *deriver) decisionEvent(
	l *lookups,
	decision *agent.AgentDecision,
) (*aiaudit.AIAuditEvent, error) {
	if decision.ProposalID == nil || decision.ProposalID.IsNil() {
		return nil, nil //nolint:nilnil // an exception's decision is not a proposal's
	}

	proposal := l.proposals[*decision.ProposalID]
	o := owner{purpose: aiaudit.PurposeLive}
	if proposal != nil {
		o = l.ownerOf(string(agent.RunOwnerAgentRun), proposal.RunID)
	}

	event := d.base(&o)
	event.SourceKey = sourceKey("decision", decision.ID.String())
	event.Kind = aiaudit.KindProposalDecided
	event.Outcome = decisionOutcome(decision.Decision)
	event.OccurredAt = decision.CreatedAt
	event.WindowStart = decision.CreatedAt
	event.WindowEnd = decision.CreatedAt
	event.DecisionID = decision.ID
	event.ProposalID = *decision.ProposalID
	event.DecidedByUserID = decision.DecidedByUserID
	event.PrincipalType = aiaudit.PrincipalUser
	event.PrincipalID = decision.DecidedByUserID.String()
	event.Reason = d.redactor.Text(decision.ReasonCode, aiaudit.MaxReasonLength)
	event.Simulated = false
	if decision.TraceID != "" {
		event.TraceID = decision.TraceID
	}
	event.Reconstructed = decision.TraceID == "" || proposal == nil

	toolName := ""
	if proposal != nil {
		toolName = proposal.ToolName
		event.ToolName = proposal.ToolName
		event.Tier = string(proposal.AutonomyTier)
		event.HeldBy = slices.Clone(proposal.HeldBy)
		event.EgressClass = string(proposal.EgressClass)
		event.StepKey = proposal.StepKey
		if proposal.PlanID != nil {
			event.PlanID = *proposal.PlanID
		}
		withTaint(event, proposal.Tainted, proposal.Taint)
		if proposal.TargetResource != "" {
			event.EntityType = proposal.TargetResource
			event.EntityID = proposal.TargetID.String()
		}
		if policy, ok := d.redactor.Policy(proposal.ToolName); ok {
			event.ToolEffect = string(policy.EffectiveEffect())
		}
	}
	if err := d.recordArguments(event, toolName, decision.Modifications); err != nil {
		return nil, err
	}

	return event, nil
}

func decisionOutcome(decision agent.DecisionType) aiaudit.Outcome {
	switch decision {
	case agent.DecisionModified:
		return aiaudit.OutcomeModified
	case agent.DecisionRejected:
		return aiaudit.OutcomeRejected
	case agent.DecisionAccepted:
		return aiaudit.OutcomeAccepted
	default:
		return aiaudit.OutcomeAccepted
	}
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}

	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func stringOf(value any) string {
	text, _ := value.(string)

	return text
}

func int64Of(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
