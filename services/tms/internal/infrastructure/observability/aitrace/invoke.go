package aitrace

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type InvokeAgent struct {
	Anchor         Anchor
	AgentID        pulid.ID
	AgentName      string
	AgentVersion   *int64
	OwnerKind      string
	OwnerID        pulid.ID
	RunID          pulid.ID
	TurnID         pulid.ID
	ConversationID pulid.ID
	DelegateCallID string
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Trigger        string
	Status         string
	ErrorType      string
	InputTokens    int64
	OutputTokens   int64
	CostUSD        *decimal.Decimal
	ToolCalls      int
	Tainted        bool
	TaintSources   []string
	Simulation     bool
	Purpose        string
	Start          time.Time
	End            time.Time
	Origin         string
	Links          []trace.Link
}

func EmitInvokeAgent(ctx context.Context, spec *InvokeAgent) {
	if spec == nil || !spec.Anchor.IsValid() {
		return
	}

	purpose := spec.Purpose
	if purpose == "" {
		purpose = PurposeLive
	}

	attrs := make([]attribute.KeyValue, 0, 26)
	attrs = append(attrs,
		GenAIOperationName.String(OperationInvokeAgent),
		AIStatus.String(spec.Status),
		AITokensInput.Int64(spec.InputTokens),
		AITokensOutput.Int64(spec.OutputTokens),
		AIToolCalls.Int(spec.ToolCalls),
		AITainted.Bool(spec.Tainted),
		AISimulation.Bool(spec.Simulation),
		AIPurpose.String(purpose),
	)
	attrs = appendString(attrs, GenAIAgentID, spec.AgentID.String())
	attrs = appendString(attrs, GenAIAgentName, spec.AgentName)
	attrs = appendString(attrs, GenAIConversationID, spec.ConversationID.String())
	attrs = appendString(attrs, UserID, spec.UserID.String())
	attrs = appendString(attrs, AIOwnerKind, spec.OwnerKind)
	attrs = appendString(attrs, AIOwnerID, spec.OwnerID.String())
	attrs = appendString(attrs, AIRunID, spec.RunID.String())
	attrs = appendString(attrs, AITurnID, spec.TurnID.String())
	attrs = appendString(attrs, AIDelegateCallID, spec.DelegateCallID)
	attrs = appendString(attrs, AITrigger, spec.Trigger)
	attrs = appendString(attrs, TenantOrganizationID, spec.OrganizationID.String())
	attrs = appendString(attrs, TenantBusinessUnitID, spec.BusinessUnitID.String())
	if spec.AgentVersion != nil {
		attrs = append(attrs, AIAgentVersion.Int64(*spec.AgentVersion))
	}
	if spec.CostUSD != nil {
		attrs = append(attrs, AICostUSD.Float64(spec.CostUSD.InexactFloat64()))
	}
	if len(spec.TaintSources) > 0 {
		attrs = append(attrs, AITaintSources.StringSlice(spec.TaintSources))
	}

	status := codes.Unset
	if spec.ErrorType != "" {
		attrs = append(attrs, ErrorType.String(spec.ErrorType))
		status = codes.Error
	}

	links := make([]trace.Link, 0, len(spec.Links)+1)
	if link, ok := LinkFromTraceparent(spec.Origin); ok {
		links = append(links, link)
	}
	links = append(links, spec.Links...)

	EmitRoot(ctx, spec.Anchor, &RootSpec{
		Name:              spanName(OperationInvokeAgent, spec.AgentName),
		Start:             spec.Start,
		End:               spec.End,
		Attrs:             attrs,
		Links:             links,
		Status:            status,
		StatusDescription: spec.ErrorType,
	})
}
