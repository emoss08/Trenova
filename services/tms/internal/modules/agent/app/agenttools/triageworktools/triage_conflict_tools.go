package triageworktools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/inboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageconflictservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	paramConflictID         = "conflictId"
	paramConflictResolution = "resolution"
	maxConflictID           = 200
)

var triageConflictResolutions = agenttoolschema.Source(
	"inboundMessage.conflictResolution",
	inboundmessage.AllConflictResolutions(),
)

type triageConflicts interface {
	Resolve(
		ctx context.Context,
		req *triageconflictservice.ResolveRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanResolve(
		ctx context.Context,
		req *triageconflictservice.ResolveRequest,
	) (*triageconflictservice.ConflictPlan, error)
	Reopen(
		ctx context.Context,
		req *triageconflictservice.ReopenRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanReopen(
		ctx context.Context,
		req *triageconflictservice.ReopenRequest,
	) (*triageconflictservice.ConflictPlan, error)
}

type conflictView struct {
	Kind       string `json:"kind"`
	Resolution string `json:"resolution,omitempty"`
	ResolvedAt int64  `json:"resolvedAt,omitempty"`
	ResolvedBy string `json:"resolvedBy,omitempty"`
}

func conflictViewOf(conflict *inboundmessage.Conflict) *conflictView {
	return &conflictView{
		Kind:       string(conflict.Kind),
		Resolution: string(conflict.Resolution),
		ResolvedAt: conflict.ResolvedAt,
		ResolvedBy: inboundtools.IDText(conflict.ResolvedBy),
	}
}

func conflictLabel(kind inboundmessage.ConflictKind) string {
	if kind == inboundmessage.ConflictOverlappingLoad {
		return "Repeated load"
	}

	return "Duplicate document"
}

func renderConflictPlan(
	summary string,
	plan *triageconflictservice.ConflictPlan,
) (*agentpreview.ToolPreview, error) {
	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceInboundMessage,
			ID:       plan.Message.ID,
			Label: conflictLabel(plan.Before.Kind) + " on " +
				inboundtools.DescribeMessage(plan.Message),
			Version: changepreview.PreviewVersion(plan.Message.Version),
		},
		conflictViewOf(&plan.Before),
		conflictViewOf(&plan.After),
		toolpreview.WithRefs(map[string]permission.Resource{"resolvedBy": permission.ResourceUser}),
		toolpreview.Labels(map[string]string{
			"kind":       "Conflict",
			"resolution": "Settled as",
			"resolvedAt": "Settled at",
			"resolvedBy": "Settled by",
		}),
		toolpreview.Volatile("resolvedAt"),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}

func NewResolveInboundConflictTool(p inboundtools.TriageToolParams) agenttoolports.AgentTool {
	return newResolveInboundConflictTool(p.Conflicts)
}

func newResolveInboundConflictTool(conflicts triageConflicts) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:     "inbox_resolve_conflict",
			Artifact: inboundtools.MessageEntity,
			Description: "Settle a conflict Triage found on a message. A duplicate document: " +
				"ReplaceExisting files the new one in place of the one on file, KeepBoth " +
				"keeps both. A repeated load: SkipDuplicate when the load is already " +
				"entered, ConfirmNew when it is a separate load. inbox_get_inbound_message lists " +
				"each conflict with its id and what it compares. Settle a document conflict " +
				"before filing the attachment.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Decides whether a document on file is replaced or a load is a " +
				"duplicate, which later filing and booking follow, so a person approves it.",
			Properties: map[string]any{
				inboundtools.ParamMessageID: inboundtools.MessageIDProperty(
					"The message, from inbox_list_inbound_messages, newest first or by query.",
				),
				paramConflictID: agenttoolschema.IDText(
					"The conflict, from the conflicts inbox_get_inbound_message lists for the message.",
				),
				paramConflictResolution: agenttoolschema.Enum(
					"How to settle it: ReplaceExisting or KeepBoth for a duplicate document, "+
						"SkipDuplicate or ConfirmNew for a repeated load.",
					triageConflictResolutions,
				),
			},
			Required: []string{
				inboundtools.ParamMessageID,
				paramConflictID,
				paramConflictResolution,
			},
			SearchTerms: []string{"duplicate document", "duplicate load", "replace on file"},
			Recipe:      []string{inboundtools.ToolGetMessage, "inbox_resolve_conflict"},
			Target:      inboundtools.TargetMessage,
		},
		toolspec.ReceivablePlan[
			*triageconflictservice.ResolveRequest,
			*triageconflictservice.ConflictPlan,
		]{
			Request: resolveConflictRequest,
			Plan: func(
				ctx context.Context,
				req *triageconflictservice.ResolveRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*triageconflictservice.ConflictPlan, error) {
				return conflicts.PlanResolve(ctx, req)
			},
			Refused: func(*triageconflictservice.ResolveRequest) string {
				return "Would settle a conflict on an inbound message."
			},
			Render: func(
				req *triageconflictservice.ResolveRequest,
				plan *triageconflictservice.ConflictPlan,
			) (*agentpreview.ToolPreview, error) {
				return renderConflictPlan(
					"Would settle the "+strings.ToLower(conflictLabel(plan.Before.Kind))+
						" conflict as "+string(req.Resolution)+".",
					plan,
				)
			},
			Run: func(
				ctx context.Context,
				req *triageconflictservice.ResolveRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				message, err := conflicts.Resolve(ctx, req)
				if err != nil {
					return nil, err
				}

				return inboundtools.MessageResult("settled a conflict on", message), nil
			},
		},
	)
}

func resolveConflictRequest(
	params *agenttoolports.ToolExecuteParams,
) (*triageconflictservice.ResolveRequest, error) {
	messageID, err := toolparams.RequirePulid(
		params.Params,
		inboundtools.ParamMessageID,
	)
	if err != nil {
		return nil, err
	}
	conflictID, err := toolparams.RequireBoundedText(params.Params, paramConflictID,
		maxConflictID)
	if err != nil {
		return nil, err
	}
	resolution, err := toolparams.RequireEnum(params.Params, paramConflictResolution,
		triageConflictResolutions.Values)
	if err != nil {
		return nil, err
	}

	return &triageconflictservice.ResolveRequest{
		TenantInfo: inboundtools.TenantOnly(params),
		ActorID:    inboundtools.ActorOf(params),
		MessageID:  messageID,
		ConflictID: conflictID,
		Resolution: resolution,
	}, nil
}

func NewReopenInboundConflictTool(p inboundtools.TriageToolParams) agenttoolports.AgentTool {
	return newReopenInboundConflictTool(p.Inbox, p.Conflicts)
}

func newReopenInboundConflictTool(
	inbox inboundtools.MessageReader,
	conflicts triageConflicts,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:     "inbox_reopen_conflict",
			Artifact: inboundtools.MessageEntity,
			Description: "Reopen a conflict on a message that was settled the wrong way, so " +
				"it can be settled again with inbox_resolve_conflict. A document conflict " +
				"whose attachment is already filed is refused: unfile it first with " +
				"inbox_unfile_attachment. inbox_get_inbound_message lists each conflict.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundtools.MessageCondition(inbox),
			Reversible:  true,
			Rationale: "Takes back how a conflict on an inbound message was settled; nothing " +
				"filed or booked changes.",
			Properties: map[string]any{
				inboundtools.ParamMessageID: inboundtools.MessageIDProperty(
					"The message, from inbox_list_inbound_messages, newest first or by query.",
				),
				paramConflictID: agenttoolschema.IDText(
					"The settled conflict, from the conflicts inbox_get_inbound_message lists for " +
						"the message.",
				),
			},
			Required:    []string{inboundtools.ParamMessageID, paramConflictID},
			SearchTerms: []string{"reopen conflict", "undo duplicate decision"},
			Recipe:      []string{inboundtools.ToolGetMessage, "inbox_reopen_conflict"},
			Target:      inboundtools.TargetMessage,
		},
		toolspec.ReceivablePlan[
			*triageconflictservice.ReopenRequest,
			*triageconflictservice.ConflictPlan,
		]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*triageconflictservice.ReopenRequest, error) {
				messageID, err := toolparams.RequirePulid(
					params.Params,
					inboundtools.ParamMessageID,
				)
				if err != nil {
					return nil, err
				}
				conflictID, err := toolparams.RequireBoundedText(params.Params, paramConflictID,
					maxConflictID)
				if err != nil {
					return nil, err
				}

				return &triageconflictservice.ReopenRequest{
					TenantInfo: inboundtools.TenantOnly(params),
					ActorID:    inboundtools.ActorOf(params),
					MessageID:  messageID,
					ConflictID: conflictID,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *triageconflictservice.ReopenRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*triageconflictservice.ConflictPlan, error) {
				return conflicts.PlanReopen(ctx, req)
			},
			Refused: func(*triageconflictservice.ReopenRequest) string {
				return "Would reopen a conflict on an inbound message."
			},
			Render: func(
				_ *triageconflictservice.ReopenRequest,
				plan *triageconflictservice.ConflictPlan,
			) (*agentpreview.ToolPreview, error) {
				return renderConflictPlan(
					"Would reopen the "+strings.ToLower(conflictLabel(plan.Before.Kind))+
						" conflict, settled as "+string(plan.Before.Resolution)+", so it can "+
						"be settled again.",
					plan,
				)
			},
			Run: func(
				ctx context.Context,
				req *triageconflictservice.ReopenRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				message, err := conflicts.Reopen(ctx, req)
				if err != nil {
					return nil, err
				}

				return inboundtools.MessageResult("reopened a conflict on", message), nil
			},
		},
	)
}
