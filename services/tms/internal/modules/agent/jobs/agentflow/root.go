package agentflow

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/trace"
)

const (
	delegateFieldCallID    = "delegateCallId"
	delegateFieldStatus    = "status"
	delegateFieldToolCalls = "toolCallsUsed"
)

type RootParams struct {
	Anchor         aitrace.Anchor
	Definition     *agentdefinition.Definition
	OwnerKind      kernelports.RunStepOwnerKind
	OwnerID        pulid.ID
	RunID          pulid.ID
	TurnID         pulid.ID
	ThreadID       pulid.ID
	DelegateCallID string
	UserID         pulid.ID
	Tenant         pagination.TenantInfo
	Trigger        string
	Status         string
	Failure        *modelcallcontract.Failure
	Result         *agentports.RunResult
	Usage          *agentports.RunUsage
	ToolCalls      int
	Purpose        string
	Start          time.Time
	End            time.Time
	Origin         string
	Links          []trace.Link
}

func EmitRoot(ctx context.Context, p *RootParams) {
	if p == nil || !p.Anchor.IsValid() {
		return
	}

	spec := &aitrace.InvokeAgent{
		Anchor:         p.Anchor,
		OwnerKind:      string(p.OwnerKind),
		OwnerID:        p.OwnerID,
		RunID:          p.RunID,
		TurnID:         p.TurnID,
		ConversationID: p.ThreadID,
		DelegateCallID: p.DelegateCallID,
		UserID:         p.UserID,
		OrganizationID: p.Tenant.OrgID,
		BusinessUnitID: p.Tenant.BuID,
		Trigger:        p.Trigger,
		Status:         p.Status,
		ErrorType:      agentflowcontract.FailureKind(p.Failure),
		ToolCalls:      p.ToolCalls,
		Purpose:        p.Purpose,
		Start:          p.Start,
		End:            p.End,
		Origin:         p.Origin,
		Links:          p.Links,
	}
	if definition := p.Definition; definition != nil {
		version := definition.Version
		spec.AgentID = definition.ID
		spec.AgentName = definition.Name
		spec.AgentVersion = &version
		spec.Simulation = definition.SimulationMode
	}

	usage := p.Usage
	if result := p.Result; result != nil {
		if usage == nil {
			usage = result.Usage
		}
		if spec.ToolCalls == 0 {
			spec.ToolCalls = result.ToolCallsUsed
		}
		spec.Tainted = result.Taint.Tainted()
		for _, source := range result.Taint.Sources() {
			spec.TaintSources = append(spec.TaintSources, string(source))
		}
	}
	if usage != nil {
		spec.InputTokens = usage.InputTokens
		spec.OutputTokens = usage.OutputTokens
		spec.CostUSD = usage.CostUSD
	}

	aitrace.EmitInvokeAgent(ctx, spec)
}

func StartedAt(times ...int64) time.Time {
	var earliest int64
	for _, at := range times {
		if at > 0 && (earliest == 0 || at < earliest) {
			earliest = at
		}
	}
	if earliest == 0 {
		return time.Time{}
	}

	return time.Unix(earliest, 0)
}

type DelegateSpan struct {
	Start     time.Time
	End       time.Time
	Status    string
	ToolCalls int
}

func DelegateSpans(events []temporaltype.StreamItem) map[string]DelegateSpan {
	spans := make(map[string]DelegateSpan)
	for _, event := range events {
		if event.Event != agentports.AssistantEventDelegateStarted &&
			event.Event != agentports.AssistantEventDelegateFinished {
			continue
		}
		fields := eventFields(event.Data)
		callID, _ := fields[delegateFieldCallID].(string)
		if callID == "" {
			continue
		}

		span := spans[callID]
		at := time.Unix(event.At, 0)
		if event.At == 0 {
			at = time.Time{}
		}
		if event.Event == agentports.AssistantEventDelegateStarted {
			span.Start = at
		} else {
			span.End = at
			span.Status, _ = fields[delegateFieldStatus].(string)
			span.ToolCalls = intutils.IntValue(fields[delegateFieldToolCalls])
		}
		spans[callID] = span
	}

	return spans
}

func eventFields(data any) map[string]any {
	switch typed := data.(type) {
	case map[string]any:
		return typed
	case agentports.AssistantDelegateStartedEvent:
		return map[string]any{delegateFieldCallID: typed.DelegateCallID}
	case *agentports.AssistantDelegateStartedEvent:
		return map[string]any{delegateFieldCallID: typed.DelegateCallID}
	case agentports.AssistantDelegateFinishedEvent:
		return delegateFinishedFields(&typed)
	case *agentports.AssistantDelegateFinishedEvent:
		return delegateFinishedFields(typed)
	default:
		return nil
	}
}

func delegateFinishedFields(event *agentports.AssistantDelegateFinishedEvent) map[string]any {
	return map[string]any{
		delegateFieldCallID:    event.DelegateCallID,
		delegateFieldStatus:    string(event.Status),
		delegateFieldToolCalls: event.ToolCallsUsed,
	}
}
