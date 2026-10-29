package inboundtools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageconflictservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagepeekservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	paramMessageIDs        = "messageIds"
	maxTriageSelection     = 50
	inboundMessageEntity   = "inbound_message"
	toolGetInboundMessage  = "inbox_get_inbound_message"
	toolListInboundMessage = "inbox_list_inbound_messages"
	toolListAssignees      = "inbox_list_triage_assignees"
	fieldMatchReason       = "matchReason"
)

type triageToolParams struct {
	fx.In

	Inbox     *inboundmessageservice.Service
	Work      *triageworkservice.Service
	Replies   *inboundreplyservice.Service
	Conflicts *triageconflictservice.Service
	Peeks     *triagepeekservice.Service
}

type triageQueue interface {
	inboundMessageReader
	Assign(
		ctx context.Context,
		req *inboundmessageservice.AssignRequest,
	) (*inboundmessageports.BulkResult, error)
	PlanAssign(
		ctx context.Context,
		req *inboundmessageservice.AssignRequest,
	) (*inboundmessageservice.QueuePlan, error)
	Reopen(
		ctx context.Context,
		req *inboundmessageservice.ReopenRequest,
	) (*inboundmessageports.BulkResult, error)
	PlanReopen(
		ctx context.Context,
		req *inboundmessageservice.ReopenRequest,
	) (*inboundmessageservice.QueuePlan, error)
	Snooze(
		ctx context.Context,
		req *inboundmessageservice.SnoozeRequest,
	) (*inboundmessageports.BulkResult, error)
	PlanSnooze(
		ctx context.Context,
		req *inboundmessageservice.SnoozeRequest,
	) (*inboundmessageservice.QueuePlan, error)
	Wake(
		ctx context.Context,
		req *inboundmessageservice.MessageActionRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanWake(
		ctx context.Context,
		req *inboundmessageservice.MessageActionRequest,
	) (*inboundmessageservice.QueuePlan, error)
	StopWaiting(
		ctx context.Context,
		req *inboundmessageservice.MessageActionRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanStopWaiting(
		ctx context.Context,
		req *inboundmessageservice.MessageActionRequest,
	) (*inboundmessageservice.QueuePlan, error)
	WaitForReply(
		ctx context.Context,
		req *inboundmessageservice.WaitForReplyRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanWaitForReply(
		ctx context.Context,
		req *inboundmessageservice.WaitForReplyRequest,
	) (*inboundmessageservice.QueuePlan, error)
	Unlink(
		ctx context.Context,
		req *inboundmessageservice.UnlinkRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanUnlink(
		ctx context.Context,
		req *inboundmessageservice.UnlinkRequest,
	) (*inboundmessageservice.QueuePlan, error)
	HandOff(
		ctx context.Context,
		req *inboundmessageservice.HandOffRequest,
	) (*inboundmessageports.HandOffResult, error)
	PlanHandOff(
		ctx context.Context,
		req *inboundmessageservice.HandOffRequest,
	) (*inboundmessageservice.QueuePlan, error)
}

var errNothingInTheQueueChanges = errors.New(
	"none of these messages would change: each is already as asked, settled where that " +
		"matters, or not in this organization",
)

func messageIDsProperty(description string) map[string]any {
	return agenttoolschema.RecordIDs(permission.ResourceInboundMessage, description,
		maxTriageSelection)
}

func messageIDProperty(description string) map[string]any {
	return agenttoolschema.RecordIDText(permission.ResourceInboundMessage, description)
}

func triageSelection(
	params *agenttoolports.ToolExecuteParams,
) (inboundmessageservice.Selection, error) {
	ids, err := toolparams.RequirePulidSlice(params.Params, paramMessageIDs, maxTriageSelection)
	if err != nil {
		return inboundmessageservice.Selection{}, err
	}

	return inboundmessageservice.Selection{IDs: ids}, nil
}

func targetMessage(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramMessageID, permission.ResourceInboundMessage)
}

func tenantOnly(params *agenttoolports.ToolExecuteParams) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: params.OrganizationID, BuID: params.BusinessUnitID}
}

func actorOf(params *agenttoolports.ToolExecuteParams) pulid.ID {
	return params.Actor.UserID
}

func messageAction(
	params *agenttoolports.ToolExecuteParams,
) (*inboundmessageservice.MessageActionRequest, error) {
	messageID, err := toolparams.RequirePulid(params.Params, paramMessageID)
	if err != nil {
		return nil, err
	}

	return &inboundmessageservice.MessageActionRequest{
		TenantInfo: tenantOnly(params),
		ActorID:    actorOf(params),
		MessageID:  messageID,
	}, nil
}

type queueStateView struct {
	Status               string `json:"status"`
	OwnerID              string `json:"ownerId,omitempty"`
	Parked               string `json:"parked,omitempty"`
	SnoozedUntil         int64  `json:"snoozedUntil,omitempty"`
	SnoozeEvent          string `json:"snoozeEvent,omitempty"`
	SnoozeWatchID        string `json:"snoozeWatchId,omitempty"`
	SnoozeDocumentTypeID string `json:"snoozeDocumentTypeId,omitempty"`
	AwaitingUntil        int64  `json:"awaitingUntil,omitempty"`
	DueAt                int64  `json:"dueAt,omitempty"`
	MatchedShipmentID    string `json:"matchedShipmentId,omitempty"`
	MatchedCustomerID    string `json:"matchedCustomerId,omitempty"`
	MatchedCarrierID     string `json:"matchedCarrierId,omitempty"`
	MatchReason          string `json:"matchReason,omitempty"`
	ReviewNote           string `json:"reviewNote,omitempty"`
}

var queueStateRefs = map[string]permission.Resource{
	"ownerId":              permission.ResourceUser,
	"snoozeWatchId":        permission.ResourceShipment,
	"snoozeDocumentTypeId": permission.ResourceDocumentType,
	"matchedShipmentId":    permission.ResourceShipment,
	"matchedCustomerId":    permission.ResourceCustomer,
	"matchedCarrierId":     permission.ResourceCarrier,
}

var queueStateLabels = map[string]string{
	"ownerId":              "Owner",
	"parked":               "Parked",
	"snoozedUntil":         "Snoozed until",
	"snoozeEvent":          "Wakes when",
	"snoozeWatchId":        "Watching shipment",
	"snoozeDocumentTypeId": "Waiting for document",
	"awaitingUntil":        "Waits for a reply until",
	"dueAt":                "Due",
	"matchedShipmentId":    changepreview.LabelShipment,
	"matchedCustomerId":    changepreview.LabelCustomer,
	"matchedCarrierId":     changepreview.LabelCarrier,
	fieldMatchReason:       "Why",
	"reviewNote":           "Note",
}

func idText(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}

func queueViewOf(state *inboundmessageports.QueueState) *queueStateView {
	return &queueStateView{
		Status:               string(state.Status),
		OwnerID:              idText(state.OwnerID),
		Parked:               string(state.Parked),
		SnoozedUntil:         state.SnoozedUntil,
		SnoozeEvent:          string(state.SnoozeEvent),
		SnoozeWatchID:        idText(state.SnoozeWatchID),
		SnoozeDocumentTypeID: idText(state.SnoozeDocumentTypeID),
		AwaitingUntil:        state.AwaitingUntil,
		DueAt:                state.DueAt,
		MatchedShipmentID:    idText(state.MatchedShipmentID),
		MatchedCustomerID:    idText(state.MatchedCustomerID),
		MatchedCarrierID:     idText(state.MatchedCarrierID),
		MatchReason:          state.MatchReason,
		ReviewNote:           state.ReviewNote,
	}
}

func describeQueueState(state *inboundmessageports.QueueState) string {
	return describeInboundMessage(&inboundmessage.InboundMessage{
		Subject:     state.Subject,
		FromAddress: state.FromAddress,
	})
}

func renderQueuePlan(
	summary string,
	plan *inboundmessageservice.QueuePlan,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		built, err := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceInboundMessage,
				ID:       change.Before.ID,
				Label:    describeQueueState(change.Before),
			},
			queueViewOf(change.Before),
			queueViewOf(change.After),
			toolpreview.WithRefs(queueStateRefs),
			toolpreview.Labels(queueStateLabels),
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, built)
	}

	if len(changes) == 0 {
		return refusedPreview(summary+".", errNothingInTheQueueChanges), nil
	}

	preview := toolpreview.Build(summary+skippedNote(plan)+".", changes...)
	if plan.More {
		preview.Partial = true
	}

	return preview, nil
}

func countMessages(n int) string {
	if n == 1 {
		return "1 message"
	}

	return fmt.Sprintf("%d messages", n)
}

func skippedNote(plan *inboundmessageservice.QueuePlan) string {
	parts := make([]string, 0, 2)
	if plan.Unchanged > 0 {
		parts = append(parts, countMessages(plan.Unchanged)+" already as asked or not eligible")
	}
	if plan.Missing > 0 {
		parts = append(parts, countMessages(plan.Missing)+" not found")
	}
	if len(parts) == 0 {
		return ""
	}

	return " (" + strings.Join(parts, "; ") + " left alone)"
}

func messageResult(
	action string,
	message *inboundmessage.InboundMessage,
) *agentrecord.ToolExecutionResult {
	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   "inbound message",
		Name:   describeInboundMessage(message),
		IDs:    map[string]string{paramMessageID: message.ID.String()},
		Record: changepreview.RecordOf(inboundMessageEntity, message.ID),
	}
}

func bulkResult(
	action string,
	result *inboundmessageports.BulkResult,
) *agentrecord.ToolExecutionResult {
	out := &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   "inbound messages",
		Name:   countMessages(result.Affected),
		IDs:    make(map[string]string, 1),
	}
	ids := make([]string, 0, len(result.Messages))
	for _, message := range result.Messages {
		ids = append(ids, message.ID.String())
	}
	if len(ids) > 0 {
		out.IDs[paramMessageIDs] = strings.Join(ids, ",")
	}
	if len(result.Messages) == 1 {
		out.Record = changepreview.RecordOf(inboundMessageEntity, result.Messages[0].ID)
	}

	return out
}

func inboundMessageChange(
	before, after *inboundmessage.InboundMessage,
	opts ...toolpreview.Option,
) (*agentpreview.RecordChange, error) {
	return toolpreview.Changed(inboundMessageRecord(before), before, after, opts...)
}

func refusedPreview(summary string, err error) *agentpreview.ToolPreview {
	return changepreview.WarnWouldFail(toolpreview.Build(summary), err)
}
