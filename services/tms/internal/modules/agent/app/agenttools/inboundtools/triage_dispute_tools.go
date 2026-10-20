package inboundtools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramExceptionKind     = "kind"
	paramExceptionQuantity = "quantity"
	paramExceptionUnit     = "unit"
	paramExceptionNote     = "note"
	paramStopsDecision     = "decision"
	maxExceptionQuantity   = 100000
	maxExceptionUnit       = 50
	maxExceptionNote       = 500
)

var (
	triageExceptionKinds = agenttoolschema.Source(
		"inboundMessage.exceptionKind",
		inboundmessage.AllExceptionKinds(),
	)
	triageStopsDecisions = agenttoolschema.Source(
		"inboundMessage.stopsDecision",
		inboundmessage.AllStopsDecisions(),
	)
	errStopsAlreadyDecided = errors.New(
		"this dispute was already decided that way; nothing would change",
	)
)

type triageDisputes interface {
	LogException(
		ctx context.Context,
		req *triageworkservice.LogExceptionRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanLogException(
		ctx context.Context,
		req *triageworkservice.LogExceptionRequest,
	) (*triageworkservice.ExceptionPlan, error)
	DecideStops(
		ctx context.Context,
		req *triageworkservice.DecideStopsRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanDecideStops(
		ctx context.Context,
		req *triageworkservice.DecideStopsRequest,
	) (*triageworkservice.StopsPlan, error)
}

type triageStopTimes interface {
	StopTimes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		messageID pulid.ID,
	) (*inboundmessageports.StopTimes, error)
}

type exceptionHoldView struct {
	Kind     string `json:"kind"`
	Quantity int    `json:"quantity"`
	Unit     string `json:"unit"`
	Note     string `json:"note,omitempty"`
	Source   string `json:"source"`
}

type actionStateView struct {
	ExceptionKind string `json:"exceptionKind,omitempty"`
	StopsDecision string `json:"stopsDecision,omitempty"`
}

func actionStateOf(message *inboundmessage.InboundMessage) *actionStateView {
	return &actionStateView{
		ExceptionKind: string(message.ActionState.ExceptionKind),
		StopsDecision: string(message.ActionState.StopsDecision),
	}
}

func actionStateChange(before, after *inboundmessage.InboundMessage) (*agent.RecordChange, error) {
	return toolpreview.Changed(
		inboundMessageRecord(before),
		actionStateOf(before),
		actionStateOf(after),
		toolpreview.Labels(map[string]string{
			"exceptionKind": "Exception logged",
			"stopsDecision": "Detention decision",
		}),
	)
}

func NewLogInboundExceptionTool(p triageToolParams) agentports.AgentTool {
	return newLogInboundExceptionTool(p.Inbox, p.Work)
}

func newLogInboundExceptionTool(
	inbox inboundMessageReader,
	disputes triageDisputes,
) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_record_discrepancy",
			Artifact: inboundMessageEntity,
			Description: "Log a shortage, overage or damage a delivery email reports on the " +
				"message's linked shipment, as a billing hold. That hold keeps it from being " +
				"invoiced until someone releases it. Count what the sender reports, in the " +
				"units they use, and never estimate. The message must be linked to its " +
				"shipment first (inbox_link_inbound_message).",
			Resource:    permission.ResourceShipmentHold,
			Operation:   permission.OpCreate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundCondition(inbox),
			Reversible:  true,
			Rationale: "Places a billing hold on a shipment inside Trenova; nothing is sent " +
				"and a person can release it.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The delivery email, from inbox_list_inbound_messages, newest first or by query.",
				),
				paramExceptionKind: agenttoolschema.Enum(
					"What the delivery reported.",
					triageExceptionKinds,
				),
				paramExceptionQuantity: map[string]any{
					toolschema.KeyType:        toolschema.TypeInteger,
					toolschema.KeyDescription: "How many, as the sender counted them.",
					toolschema.KeyMinimum:     1,
					toolschema.KeyMaximum:     maxExceptionQuantity,
				},
				paramExceptionUnit: common.StringProperty(
					"What was counted: pallets, cases, pieces.",
					maxExceptionUnit,
				),
				paramExceptionNote: common.StringProperty(
					"Optional: what the sender said about it, for the person who clears the hold.",
					maxExceptionNote,
				),
			},
			Required: []string{
				paramMessageID,
				paramExceptionKind,
				paramExceptionQuantity,
				paramExceptionUnit,
			},
			SearchTerms: []string{"shortage", "overage", "damaged goods", "OS&D", "billing hold"},
			Recipe:      []string{toolGetInboundMessage, "inbox_record_discrepancy"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[
			*triageworkservice.LogExceptionRequest,
			*triageworkservice.ExceptionPlan,
		]{
			Request: logExceptionRequest,
			Plan: func(
				ctx context.Context,
				req *triageworkservice.LogExceptionRequest,
				_ *agentports.ToolExecuteParams,
			) (*triageworkservice.ExceptionPlan, error) {
				return disputes.PlanLogException(ctx, req)
			},
			Refused: func(*triageworkservice.LogExceptionRequest) string {
				return "Would log a delivery exception as a billing hold."
			},
			Render: renderLoggedException,
			Run: func(
				ctx context.Context,
				req *triageworkservice.LogExceptionRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				message, err := disputes.LogException(ctx, req)
				if err != nil {
					return nil, err
				}

				return messageResult("logged an exception from", message), nil
			},
		},
	)
}

func renderLoggedException(
	_ *triageworkservice.LogExceptionRequest,
	plan *triageworkservice.ExceptionPlan,
) (*agent.ToolPreview, error) {
	hold, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceShipmentHold,
			Label:    "Billing hold: " + plan.Summary,
		},
		&exceptionHoldView{
			Kind:     string(plan.Hold.Kind),
			Quantity: plan.Hold.Quantity,
			Unit:     plan.Hold.Unit,
			Note:     plan.Hold.Note,
			Source:   plan.Hold.Source,
		},
	)
	if err != nil {
		return nil, err
	}
	logged, err := actionStateChange(plan.Message, plan.After)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(
		"Would hold the linked shipment from billing for "+plan.Summary+".",
		hold,
		logged,
	), nil
}

func logExceptionRequest(
	params *agentports.ToolExecuteParams,
) (*triageworkservice.LogExceptionRequest, error) {
	messageID, err := common.RequirePulid(params.Params, paramMessageID)
	if err != nil {
		return nil, err
	}
	kind, err := common.RequireEnum(params.Params, paramExceptionKind, triageExceptionKinds.Values)
	if err != nil {
		return nil, err
	}
	quantity, err := common.RequireIntInRange(params.Params, paramExceptionQuantity, 1,
		maxExceptionQuantity)
	if err != nil {
		return nil, err
	}
	unit, err := common.RequireBoundedText(params.Params, paramExceptionUnit, maxExceptionUnit)
	if err != nil {
		return nil, err
	}
	note, err := common.BoundedText(params.Params, paramExceptionNote, maxExceptionNote)
	if err != nil {
		return nil, err
	}

	return &triageworkservice.LogExceptionRequest{
		TenantInfo: tenantOnly(params),
		UserID:     actorOf(params),
		MessageID:  messageID,
		Kind:       kind,
		Quantity:   quantity,
		Unit:       unit,
		Note:       note,
	}, nil
}

type stopsDecisionPlan struct {
	plan  *triageworkservice.StopsPlan
	times *inboundmessageports.StopTimes
}

func NewSettleDetentionDisputeTool(p triageToolParams) agentports.AgentTool {
	return newSettleDetentionDisputeTool(p.Work, p.Peeks)
}

func newSettleDetentionDisputeTool(
	disputes triageDisputes,
	stops triageStopTimes,
) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "detention_resolve_dispute",
			Artifact: inboundMessageEntity,
			Description: "Waive or stand by a detention charge a customer disputes by email, " +
				"once the stop times are checked, and have Triage draft the answer for a " +
				"person to send. Waive when the times do not support the charge; stand by " +
				"it when arrival, departure and free time hold up. Check them first with " +
				"detention_get_occurrence or shipment_get_tracking. A waived charge already " +
				"invoiced is credited.",
			Resource:    permission.ResourceDetention,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressMoney,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierPropose,
			Rationale: "Gives up or keeps detention revenue the customer disputes, and may " +
				"credit an invoice; only a person approves it.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The detention dispute, from inbox_list_inbound_messages, " +
						"newest first or by query.",
				),
				paramStopsDecision: agenttoolschema.Enum(
					"Waive gives the charge up; StandBy keeps it.",
					triageStopsDecisions,
				),
			},
			Required:    []string{paramMessageID, paramStopsDecision},
			SearchTerms: []string{"disputed detention", "contested detention"},
			Recipe: []string{
				toolGetInboundMessage,
				"detention_get_occurrence",
				"detention_resolve_dispute",
			},
			Target: targetMessage,
		},
		common.ReceivablePlan[*triageworkservice.DecideStopsRequest, *stopsDecisionPlan]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*triageworkservice.DecideStopsRequest, error) {
				messageID, err := common.RequirePulid(params.Params, paramMessageID)
				if err != nil {
					return nil, err
				}
				decision, err := common.RequireEnum(params.Params, paramStopsDecision,
					triageStopsDecisions.Values)
				if err != nil {
					return nil, err
				}

				return &triageworkservice.DecideStopsRequest{
					TenantInfo: tenantOnly(params),
					UserID:     actorOf(params),
					Actor:      params.Actor,
					MessageID:  messageID,
					Decision:   decision,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *triageworkservice.DecideStopsRequest,
				_ *agentports.ToolExecuteParams,
			) (*stopsDecisionPlan, error) {
				plan, err := disputes.PlanDecideStops(ctx, req)
				if err != nil {
					return nil, err
				}
				times, err := stops.StopTimes(ctx, req.TenantInfo, req.MessageID)
				if err != nil {
					return nil, err
				}

				return &stopsDecisionPlan{plan: plan, times: times}, nil
			},
			Refused: func(*triageworkservice.DecideStopsRequest) string {
				return "Would decide a detention dispute."
			},
			Render: renderStopsDecision,
			Run: func(
				ctx context.Context,
				req *triageworkservice.DecideStopsRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				message, err := disputes.DecideStops(ctx, req)
				if err != nil {
					return nil, err
				}

				return messageResult("decided the detention dispute of", message), nil
			},
		},
	)
}

func renderStopsDecision(
	req *triageworkservice.DecideStopsRequest,
	decided *stopsDecisionPlan,
) (*agent.ToolPreview, error) {
	verb := "stand by"
	if req.Decision == inboundmessage.StopsDecisionWaive {
		verb = "waive"
	}
	summary := "Would " + verb + " the disputed detention charge"
	if decided.plan.Unchanged {
		return refusedPreview(summary+".", errStopsAlreadyDecided), nil
	}

	decision, err := actionStateChange(decided.plan.Message, decided.plan.After)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{decision}

	if waivesBilledDetention(req.Decision, decided.times) {
		money, moneyErr := detentionMoney(decided.times)
		if moneyErr != nil {
			return nil, moneyErr
		}
		changes = append(changes, money)
	}

	return toolpreview.Build(
		summary+stopsEvidence(decided.times)+", and have Triage draft the answer for a person "+
			"to send.",
		changes...,
	), nil
}

func stopsEvidence(times *inboundmessageports.StopTimes) string {
	if times == nil {
		return " (no detention is on record for the linked shipment)"
	}

	parts := make([]string, 0, 3)
	if times.StopName != "" {
		parts = append(parts, "at "+times.StopName)
	}
	parts = append(parts, fmt.Sprintf("%d billed minutes", times.BilledDetentionMinutes))
	if times.BilledDetentionAmount != nil {
		parts = append(parts, strings.TrimSpace(*times.BilledDetentionAmount+" "+times.Currency))
	}

	return " (" + strings.Join(parts, ", ") + ")"
}

type detentionChargeView struct {
	BillableAmount string `json:"billableAmount"`
}

func waivesBilledDetention(
	decision inboundmessage.StopsDecision,
	times *inboundmessageports.StopTimes,
) bool {
	return decision == inboundmessage.StopsDecisionWaive && times != nil &&
		times.BilledDetentionAmount != nil && times.OccurrenceID.IsNotNil()
}

func detentionMoney(times *inboundmessageports.StopTimes) (*agent.RecordChange, error) {
	amount, err := decimal.NewFromString(*times.BilledDetentionAmount)
	if err != nil {
		return nil, fmt.Errorf("read the billed detention amount: %w", err)
	}

	label := "Detention at " + times.StopName
	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceDetention,
			ID:       times.OccurrenceID,
			Label:    label,
		},
		&detentionChargeView{BillableAmount: amount.StringFixed(2)},
		&detentionChargeView{BillableAmount: decimal.Zero.StringFixed(2)},
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		toolpreview.MoneyBlock(times.Currency, agent.MoneyLine{
			Label:  label,
			Before: common.KnownAmount(amount),
			After:  common.KnownAmount(decimal.Zero),
		}),
		toolpreview.SensitiveAs("billableAmount"),
	)

	return change, nil
}
