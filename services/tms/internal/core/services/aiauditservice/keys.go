package aiauditservice

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domainvalidation"
)

// The source keys each row can stand for, worked out from the row alone, so a
// row whose trail rows are all written is dropped before anything is read to
// explain it.

func runKeys(run *agent.AgentRun) []string {
	keys := []string{sourceKey("run", run.ID.String(), "started")}
	if run.CompletedAt != nil {
		keys = append(keys, sourceKey("run", run.ID.String(), "ended"))
	}

	return keys
}

func turnKeys(turn *conversation.AssistantTurn) []string {
	keys := []string{sourceKey("turn", turn.ID.String(), "started")}
	if turn.Status.Terminal() {
		keys = append(keys, sourceKey("turn", turn.ID.String(), "ended"))
	}

	return keys
}

func proposalKeys(proposal *agent.AgentProposal) []string {
	keys := []string{sourceKey("proposal", proposal.ID.String(), "filed")}
	if proposal.SimulatedAt != nil || proposal.ExecutedAt != nil || proposal.ExecutionError != "" ||
		proposal.Status == agent.ProposalStatusExecuted ||
		proposal.Status == agent.ProposalStatusExecutionFailed {
		keys = append(keys, sourceKey("proposal", proposal.ID.String(), "executed"))
	}
	if proposal.Status == agent.ProposalStatusExpired {
		keys = append(keys, sourceKey("proposal", proposal.ID.String(), "expired"))
	}

	return keys
}

func stepKey(step *agent.AgentRunStep) string {
	return sourceKey("step", step.OwnerID.String(), step.StepKey)
}

func candidateKeys(rows *tenantRows) []string {
	keys := make([]string, 0,
		2*len(rows.runs)+2*len(rows.turns)+len(rows.usage)+len(rows.steps)+
			len(rows.events)+3*len(rows.proposals)+len(rows.decisions))

	for _, run := range rows.runs {
		keys = append(keys, runKeys(run)...)
	}
	for _, turn := range rows.turns {
		keys = append(keys, turnKeys(turn)...)
	}
	for _, record := range rows.usage {
		keys = append(keys, sourceKey("usage", record.ID.String()))
	}
	for _, step := range rows.steps {
		keys = append(keys, stepKey(step))
	}
	for _, event := range rows.events {
		keys = append(keys, sourceKey("event", event.ID.String()))
	}
	for _, proposal := range rows.proposals {
		keys = append(keys, proposalKeys(proposal)...)
	}
	for _, decision := range rows.decisions {
		keys = append(keys, sourceKey("decision", decision.ID.String()))
	}

	return keys
}

func allHeld(existing map[string]struct{}, keys []string) bool {
	for _, key := range keys {
		if _, ok := existing[key]; !ok {
			return false
		}
	}

	return true
}

// pendingRows keeps the rows that still have a trail row to write.
func pendingRows(rows *tenantRows, existing map[string]struct{}) *tenantRows {
	pending := newTenantRows(rows.tenant)

	for id, run := range rows.runs {
		if !allHeld(existing, runKeys(run)) {
			pending.runs[id] = run
		}
	}
	for id, turn := range rows.turns {
		if !allHeld(existing, turnKeys(turn)) {
			pending.turns[id] = turn
		}
	}
	for id, record := range rows.usage {
		if !allHeld(existing, []string{sourceKey("usage", record.ID.String())}) {
			pending.usage[id] = record
		}
	}
	for id, step := range rows.steps {
		if !allHeld(existing, []string{stepKey(step)}) {
			pending.steps[id] = step
		}
	}
	for id, event := range rows.events {
		if !allHeld(existing, []string{sourceKey("event", event.ID.String())}) {
			pending.events[id] = event
		}
	}
	for id, proposal := range rows.proposals {
		if !allHeld(existing, proposalKeys(proposal)) {
			pending.proposals[id] = proposal
		}
	}
	for id, decision := range rows.decisions {
		if !allHeld(existing, []string{sourceKey("decision", decision.ID.String())}) {
			pending.decisions[id] = decision
		}
	}

	return pending
}

// finish snapshots the names a row shows and keeps every value inside its
// column, so a row always inserts and hashes as it will be read back.
func finish(event *aiaudit.AIAuditEvent, found *lookups) {
	if name, ok := found.users[event.OnBehalfOfUserID]; ok {
		event.OnBehalfOfUserName = name
	}
	if name, ok := found.users[event.DecidedByUserID]; ok {
		event.DecidedByUserName = name
	}
	if name, ok := found.agents[event.AgentDefinitionID]; ok {
		event.AgentName = name
	}
	if event.HeldBy == nil {
		event.HeldBy = []string{}
	}
	if event.RedactedPaths == nil {
		event.RedactedPaths = []string{}
	}

	event.SourceKey = truncateRunes(event.SourceKey, aiaudit.MaxSourceKeyLength)
	event.PrincipalID = truncateRunes(event.PrincipalID, idLength)
	event.OnBehalfOfUserName = truncateRunes(
		cleanText(event.OnBehalfOfUserName),
		aiaudit.MaxNameLength,
	)
	event.DecidedByUserName = truncateRunes(
		cleanText(event.DecidedByUserName),
		aiaudit.MaxNameLength,
	)
	event.AgentName = truncateRunes(cleanText(event.AgentName), aiaudit.MaxAgentNameLength)
	event.StepKey = truncateRunes(event.StepKey, aiaudit.MaxStepKeyLength)
	event.CallID = truncateRunes(event.CallID, aiaudit.MaxCallIDLength)
	event.DelegateCallID = truncateRunes(event.DelegateCallID, aiaudit.MaxCallIDLength)
	event.ProviderKind = truncateRunes(event.ProviderKind, providerKindLength)
	event.Model = truncateRunes(event.Model, aiaudit.MaxModelLength)
	event.ToolName = truncateRunes(event.ToolName, aiaudit.MaxToolNameLength)
	event.ToolEffect = truncateRunes(event.ToolEffect, shortCodeLength)
	event.EgressClass = truncateRunes(event.EgressClass, codeLength)
	event.Tier = truncateRunes(event.Tier, codeLength)
	event.TierSource = truncateRunes(event.TierSource, codeLength)
	event.EntityType = truncateRunes(event.EntityType, aiaudit.MaxEntityTypeLength)
	event.EntityID = truncateRunes(event.EntityID, aiaudit.MaxEntityIDLength)
	event.ResultSummary = truncateRunes(event.ResultSummary, aiaudit.MaxResultSummaryLength)
	event.Reason = truncateRunes(event.Reason, aiaudit.MaxReasonLength)
	if !domainvalidation.IsTraceID(event.TraceID) {
		event.TraceID = ""
	}
	if !domainvalidation.IsSpanID(event.SpanID) {
		event.SpanID = ""
	}
	if event.OccurredAt <= 0 {
		event.OccurredAt = event.WindowEnd
	}
	if event.OwnerID.IsNil() {
		event.OwnerKind = ""
	}
	if event.OwnerKind != "" && !agent.RunOwnerKind(event.OwnerKind).IsValid() {
		event.OwnerKind = ""
	}
	if event.PrincipalType == aiaudit.PrincipalSystem && event.PrincipalID == "" {
		event.PrincipalID = serviceports.SystemPrincipalID.String()
	}
}

const (
	idLength           = 100
	providerKindLength = 50
	codeLength         = 30
	shortCodeLength    = 20
)
