package inboundtools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramOwnerID        = "ownerId"
	paramToUserID       = "toUserId"
	paramSnoozeUntil    = "until"
	paramSnoozeEvent    = "event"
	paramDocumentTypeID = "documentTypeId"
	paramLinkedRecord   = "record"
)

var (
	triageSnoozeEvents = agenttoolschema.Source(
		"inboundMessage.snoozeEvent",
		inboundmessage.AllSnoozeEvents(),
	)
	triageLinkedRecords = agenttoolschema.Source(
		"inboundMessage.linkedRecord",
		[]inboundmessageservice.LinkedRecord{
			inboundmessageservice.LinkedShipment,
			inboundmessageservice.LinkedCustomer,
			inboundmessageservice.LinkedCarrier,
		},
	)
	errInboundNotParked = errors.New(
		"this message is neither snoozed nor waiting on a reply, so it is already in the queue",
	)
	errSnoozeNeedsOne = errors.New(
		`give exactly one of "until" (a time) and "event" (something to wait for)`,
	)
)

func NewAssignInboundMessagesTool(p triageToolParams) agentports.AgentTool {
	return newAssignInboundMessagesTool(p.Inbox)
}

func newAssignInboundMessagesTool(queue triageQueue) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_assign_inbound_message",
			Artifact: inboundMessageEntity,
			Description: "Give messages waiting in the Triage queue to a person on the Triage " +
				"team, or take them off whoever has them. Choose the owner from " +
				"inbox_list_triage_assignees, which shows who can take Triage messages and how " +
				"many each already holds. Settled messages and ones the person already " +
				"owns are left alone. The new owner sees them in their own lane.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpAssign,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Changes who owns messages in the Triage queue inside Trenova; nothing " +
				"is sent and the messages themselves are unchanged.",
			Properties: map[string]any{
				paramMessageIDs: messageIDsProperty(
					"The messages, from inbox_list_inbound_messages or inbox_search_inbound_messages.",
				),
				paramOwnerID: agenttoolschema.RecordIDText(permission.ResourceUser,
					"The person to give them to, from inbox_list_triage_assignees. Leave it out to "+
						"take them off whoever has them."),
			},
			Required:    []string{paramMessageIDs},
			SearchTerms: []string{"give to", "triage owner", "unassign", "take it"},
			Recipe:      []string{toolListAssignees, "inbox_assign_inbound_message"},
		},
		common.ReceivablePlan[
			*inboundmessageservice.AssignRequest,
			*inboundmessageservice.QueuePlan,
		]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*inboundmessageservice.AssignRequest, error) {
				selection, err := triageSelection(params)
				if err != nil {
					return nil, err
				}
				owner, err := common.OptionalID(params.Params, paramOwnerID)
				if err != nil {
					return nil, err
				}

				return &inboundmessageservice.AssignRequest{
					TenantInfo:      tenantOnly(params),
					ActorID:         actorOf(params),
					Selection:       selection,
					OwnerID:         owner,
					CanAssignOthers: true,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.AssignRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundmessageservice.QueuePlan, error) {
				return queue.PlanAssign(ctx, req)
			},
			Refused: func(*inboundmessageservice.AssignRequest) string {
				return "Would change who owns messages in Triage."
			},
			Render: func(
				req *inboundmessageservice.AssignRequest,
				plan *inboundmessageservice.QueuePlan,
			) (*agent.ToolPreview, error) {
				if req.OwnerID.IsNil() {
					return renderQueuePlan(
						"Would take "+countMessages(len(plan.Changes))+" off whoever has them",
						plan,
					)
				}

				return renderQueuePlan(
					"Would give "+countMessages(len(plan.Changes))+" to a new owner",
					plan,
				)
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.AssignRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				result, err := queue.Assign(ctx, req)
				if err != nil {
					return nil, err
				}

				return bulkResult("assigned", result), nil
			},
		},
	)
}

func NewReopenInboundMessagesTool(p triageToolParams) agentports.AgentTool {
	return newReopenInboundMessagesTool(p.Inbox)
}

func newReopenInboundMessagesTool(queue triageQueue) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_reopen_inbound_message",
			Artifact: inboundMessageEntity,
			Description: "Return messages that were settled as Actioned or Ignored to the " +
				"Triage queue, waiting on a person again with a fresh due time. Use it when a " +
				"message was settled by mistake or is not finished. Messages still waiting " +
				"are left alone. Their review note is kept.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Puts settled messages back in the queue inside Trenova; nothing is " +
				"sent.",
			Properties: map[string]any{
				paramMessageIDs: messageIDsProperty(
					"The settled messages, from inbox_list_inbound_messages or " +
						"inbox_search_inbound_messages.",
				),
			},
			Required:    []string{paramMessageIDs},
			SearchTerms: []string{"return to the queue", "unsettle", "undo ignore"},
		},
		common.ReceivablePlan[
			*inboundmessageservice.ReopenRequest,
			*inboundmessageservice.QueuePlan,
		]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*inboundmessageservice.ReopenRequest, error) {
				selection, err := triageSelection(params)
				if err != nil {
					return nil, err
				}

				return &inboundmessageservice.ReopenRequest{
					TenantInfo: tenantOnly(params),
					ActorID:    actorOf(params),
					Selection:  selection,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.ReopenRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundmessageservice.QueuePlan, error) {
				return queue.PlanReopen(ctx, req)
			},
			Refused: func(*inboundmessageservice.ReopenRequest) string {
				return "Would put settled messages back in the Triage queue."
			},
			Render: func(
				_ *inboundmessageservice.ReopenRequest,
				plan *inboundmessageservice.QueuePlan,
			) (*agent.ToolPreview, error) {
				return renderQueuePlan(
					"Would put "+countMessages(len(plan.Changes))+" back in the Triage queue",
					plan,
				)
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.ReopenRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				result, err := queue.Reopen(ctx, req)
				if err != nil {
					return nil, err
				}

				return bulkResult("reopened", result), nil
			},
		},
	)
}

func NewSnoozeInboundMessagesTool(p triageToolParams) agentports.AgentTool {
	return newSnoozeInboundMessagesTool(p.Inbox)
}

func newSnoozeInboundMessagesTool(queue triageQueue) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_snooze_inbound_message",
			Artifact: inboundMessageEntity,
			Description: "Take messages waiting on a person out of the Triage queue until a " +
				"time, or until an event on their linked shipment. They return on their own " +
				"when it delivers, gets a document of a given type, or has its invoice paid. " +
				"Give until or event, not both. Waiting on an event takes every message " +
				"being linked to a shipment (inbox_link_inbound_message). To wait for the sender " +
				"to answer, use inbox_await_reply.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Hides messages from the Triage queue until a time or an event; they " +
				"return on their own and nothing is sent.",
			Properties: map[string]any{
				paramMessageIDs: messageIDsProperty(
					"The messages waiting on a person, from inbox_list_inbound_messages or " +
						"inbox_search_inbound_messages.",
				),
				paramSnoozeUntil: agenttoolschema.LocalDateTime(
					"When they return, at most 90 days out. Leave it out when waiting on " +
						"an event.",
				),
				paramSnoozeEvent: agenttoolschema.Enum(
					"What returns them instead of a time: ShipmentDelivered, "+
						"DocumentAttached (name the type) or InvoicePaid, on each message's "+
						"linked shipment.",
					triageSnoozeEvents,
				),
				paramDocumentTypeID: agenttoolschema.RecordIDText(
					permission.ResourceDocumentType,
					"With DocumentAttached only: the document type to wait for, from "+
						"document_list_types.",
				),
			},
			Required:    []string{paramMessageIDs},
			SearchTerms: []string{"snooze", "until delivered", "until the POD"},
			Recipe:      []string{toolListInboundMessage, "inbox_snooze_inbound_message"},
		},
		common.ReceivablePlan[
			*inboundmessageservice.SnoozeRequest,
			*inboundmessageservice.QueuePlan,
		]{
			Request: snoozeRequest,
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.SnoozeRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundmessageservice.QueuePlan, error) {
				return queue.PlanSnooze(ctx, req)
			},
			Refused: func(*inboundmessageservice.SnoozeRequest) string {
				return "Would snooze messages in Triage."
			},
			Render: func(
				req *inboundmessageservice.SnoozeRequest,
				plan *inboundmessageservice.QueuePlan,
			) (*agent.ToolPreview, error) {
				until := "until " + string(req.Event)
				if req.Event == "" {
					until = "until the time given"
				}

				return renderQueuePlan(
					"Would snooze "+countMessages(len(plan.Changes))+" "+until+
						"; they come back on their own",
					plan,
				)
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.SnoozeRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				result, err := queue.Snooze(ctx, req)
				if err != nil {
					return nil, err
				}

				return bulkResult("snoozed", result), nil
			},
		},
	)
}

func snoozeRequest(
	params *agentports.ToolExecuteParams,
) (*inboundmessageservice.SnoozeRequest, error) {
	selection, err := triageSelection(params)
	if err != nil {
		return nil, err
	}
	until, err := common.OptionalLocalTime(params.Params, paramSnoozeUntil,
		timeutils.LoadLocation(params.Timezone))
	if err != nil {
		return nil, err
	}
	event, given, err := common.OptionalEnum(params.Params, paramSnoozeEvent,
		triageSnoozeEvents.Values)
	if err != nil {
		return nil, err
	}
	if (until == nil) == !given {
		return nil, errSnoozeNeedsOne
	}
	documentType, err := common.OptionalID(params.Params, paramDocumentTypeID)
	if err != nil {
		return nil, err
	}

	req := &inboundmessageservice.SnoozeRequest{
		TenantInfo:     tenantOnly(params),
		ActorID:        actorOf(params),
		Selection:      selection,
		Event:          event,
		DocumentTypeID: documentType,
	}
	if until != nil {
		req.Until = *until
	}

	return req, nil
}

func NewWakeInboundMessageTool(p triageToolParams) agentports.AgentTool {
	return newWakeInboundMessageTool(p.Inbox)
}

func newWakeInboundMessageTool(queue triageQueue) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_wake_inbound_message",
			Artifact: inboundMessageEntity,
			Description: "Return a message to the Triage queue now: one that was snoozed, " +
				"or one parked waiting on the sender's reply. Use it when what it was waiting " +
				"for has happened some other way, or the wait no longer makes sense. A " +
				"message already in the queue is refused.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Reversible:  true,
			Rationale: "Puts a parked message back in front of the people working Triage; " +
				"nothing is sent.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The parked message, from inbox_list_inbound_messages or inbox_search_inbound_messages.",
				),
			},
			Required:    []string{paramMessageID},
			SearchTerms: []string{"unsnooze", "stop waiting", "return now"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[
			*inboundmessageservice.MessageActionRequest,
			*inboundmessageservice.QueuePlan,
		]{
			Request: messageAction,
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.MessageActionRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundmessageservice.QueuePlan, error) {
				parked, err := parkedAs(ctx, queue, req)
				if err != nil {
					return nil, err
				}
				switch parked {
				case inboundmessage.ParkingSnoozed:
					return queue.PlanWake(ctx, req)
				case inboundmessage.ParkingAwaitingReply:
					return queue.PlanStopWaiting(ctx, req)
				default:
					return &inboundmessageservice.QueuePlan{Unchanged: 1}, nil
				}
			},
			Refused: func(*inboundmessageservice.MessageActionRequest) string {
				return "Would bring a parked message back to the Triage queue."
			},
			Render: func(
				_ *inboundmessageservice.MessageActionRequest,
				plan *inboundmessageservice.QueuePlan,
			) (*agent.ToolPreview, error) {
				if len(plan.Changes) == 0 {
					return refusedPreview(
						"Would bring a parked message back to the Triage queue.",
						errInboundNotParked,
					), nil
				}

				return renderQueuePlan("Would bring the message back to the Triage queue now",
					plan)
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.MessageActionRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				parked, err := parkedAs(ctx, queue, req)
				if err != nil {
					return nil, err
				}

				var message *inboundmessage.InboundMessage
				switch parked {
				case inboundmessage.ParkingSnoozed:
					message, err = queue.Wake(ctx, req)
				case inboundmessage.ParkingAwaitingReply:
					message, err = queue.StopWaiting(ctx, req)
				default:
					return nil, errInboundNotParked
				}
				if err != nil {
					return nil, err
				}

				return messageResult("woken", message), nil
			},
		},
	)
}

func parkedAs(
	ctx context.Context,
	queue inboundMessageReader,
	action *inboundmessageservice.MessageActionRequest,
) (inboundmessage.Parking, error) {
	message, err := queue.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         action.MessageID,
		TenantInfo: action.TenantInfo,
	})
	if err != nil {
		return "", err
	}

	return message.Parked, nil
}

func NewWaitForInboundReplyTool(p triageToolParams) agentports.AgentTool {
	return newWaitForInboundReplyTool(p.Inbox)
}

func newWaitForInboundReplyTool(queue triageQueue) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_await_reply",
			Artifact: inboundMessageEntity,
			Description: "Park a message in Triage until its sender answers on the same " +
				"thread, or until a time if nobody does, at most 30 days out; it returns to " +
				"the queue either way. Use it after asking the sender something with " +
				"inbox_send_reply. It parks the message for the people working the " +
				"queue, not what you are doing: to be woken yourself when they answer, call " +
				"agent_create_wait with a Reply wait on the record.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Hides a message from the Triage queue until its sender replies or a " +
				"time comes; nothing is sent.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The message waiting on a person, from inbox_list_inbound_messages or " +
						"inbox_search_inbound_messages.",
				),
				paramSnoozeUntil: agenttoolschema.LocalDateTime(
					"When it returns if the sender has not replied, at most 30 days out.",
				),
			},
			Required:    []string{paramMessageID, paramSnoozeUntil},
			SearchTerms: []string{"waiting on customer", "awaiting reply", "until they answer"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[
			*inboundmessageservice.WaitForReplyRequest,
			*inboundmessageservice.QueuePlan,
		]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*inboundmessageservice.WaitForReplyRequest, error) {
				action, err := messageAction(params)
				if err != nil {
					return nil, err
				}
				until, err := common.RequireLocalTime(params.Params, paramSnoozeUntil,
					timeutils.LoadLocation(params.Timezone))
				if err != nil {
					return nil, err
				}

				return &inboundmessageservice.WaitForReplyRequest{
					TenantInfo: action.TenantInfo,
					ActorID:    action.ActorID,
					MessageID:  action.MessageID,
					Until:      until,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.WaitForReplyRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundmessageservice.QueuePlan, error) {
				return queue.PlanWaitForReply(ctx, req)
			},
			Refused: func(*inboundmessageservice.WaitForReplyRequest) string {
				return "Would park a message until its sender replies."
			},
			Render: func(
				_ *inboundmessageservice.WaitForReplyRequest,
				plan *inboundmessageservice.QueuePlan,
			) (*agent.ToolPreview, error) {
				return renderQueuePlan(
					"Would park the message until its sender replies on the thread, or until "+
						"the time given",
					plan,
				)
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.WaitForReplyRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				message, err := queue.WaitForReply(ctx, req)
				if err != nil {
					return nil, err
				}

				return messageResult("parked", message), nil
			},
		},
	)
}

func NewUnlinkInboundMessageTool(p triageToolParams) agentports.AgentTool {
	return newUnlinkInboundMessageTool(p.Inbox)
}

func newUnlinkInboundMessageTool(queue triageQueue) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_unlink_inbound_message",
			Artifact: inboundMessageEntity,
			Description: "Remove the shipment, customer or carrier a message is linked to, " +
				"when the link is wrong and the right record is not known. When the right " +
				"one is known, inbox_link_inbound_message replaces the link in one step. " +
				"inbox_get_inbound_message shows what the message is linked to.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundCondition(queue),
			Reversible:  true,
			Rationale: "Removes a link from an inbound message inside Trenova; the mailbox " +
				"decides how far it may run.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The message, from inbox_list_inbound_messages or inbox_search_inbound_messages.",
				),
				paramLinkedRecord: agenttoolschema.Enum(
					"Which link to remove.",
					triageLinkedRecords,
				),
			},
			Required:    []string{paramMessageID, paramLinkedRecord},
			SearchTerms: []string{"unlink", "remove link"},
			Recipe:      []string{toolGetInboundMessage, "inbox_unlink_inbound_message"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[
			*inboundmessageservice.UnlinkRequest,
			*inboundmessageservice.QueuePlan,
		]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*inboundmessageservice.UnlinkRequest, error) {
				action, err := messageAction(params)
				if err != nil {
					return nil, err
				}
				record, err := common.RequireEnum(params.Params, paramLinkedRecord,
					triageLinkedRecords.Values)
				if err != nil {
					return nil, err
				}

				return &inboundmessageservice.UnlinkRequest{
					TenantInfo: action.TenantInfo,
					ActorID:    action.ActorID,
					MessageID:  action.MessageID,
					Record:     record,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.UnlinkRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundmessageservice.QueuePlan, error) {
				return queue.PlanUnlink(ctx, req)
			},
			Refused: func(*inboundmessageservice.UnlinkRequest) string {
				return "Would remove a link from an inbound message."
			},
			Render: func(
				req *inboundmessageservice.UnlinkRequest,
				plan *inboundmessageservice.QueuePlan,
			) (*agent.ToolPreview, error) {
				return renderQueuePlan(
					fmt.Sprintf("Would remove the message's %s link", req.Record),
					plan,
				)
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.UnlinkRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				message, err := queue.Unlink(ctx, req)
				if err != nil {
					return nil, err
				}

				return messageResult("unlinked", message), nil
			},
		},
	)
}

func NewHandOffTriageTool(p triageToolParams) agentports.AgentTool {
	return newHandOffTriageTool(p.Inbox)
}

func newHandOffTriageTool(queue triageQueue) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_assign_queue",
			Artifact: inboundMessageEntity,
			Description: "Hand every message still waiting in the Triage queue to one person, " +
				"usually the on-call person at the end of the working day. " +
				"inbox_list_triage_assignees names the on-call person and who else can take it. " +
				"Snoozed, parked and settled messages are left alone; to move only some " +
				"messages, use inbox_assign_inbound_message.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpAssign,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Moves the whole waiting queue to one person inside Trenova; each " +
				"message can be reassigned, and nothing is sent.",
			Properties: map[string]any{
				paramToUserID: agenttoolschema.RecordIDText(permission.ResourceUser,
					"The person taking the queue, from inbox_list_triage_assignees."),
			},
			Required:    []string{paramToUserID},
			SearchTerms: []string{"end of day", "on call", "hand off the inbox", "night shift"},
			Recipe:      []string{toolListAssignees, "inbox_assign_queue"},
		},
		common.ReceivablePlan[
			*inboundmessageservice.HandOffRequest,
			*inboundmessageservice.QueuePlan,
		]{
			Request: func(
				params *agentports.ToolExecuteParams,
			) (*inboundmessageservice.HandOffRequest, error) {
				to, err := common.RequirePulid(params.Params, paramToUserID)
				if err != nil {
					return nil, err
				}

				return &inboundmessageservice.HandOffRequest{
					TenantInfo: tenantOnly(params),
					ActorID:    actorOf(params),
					ToUserID:   to,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.HandOffRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundmessageservice.QueuePlan, error) {
				return queue.PlanHandOff(ctx, req)
			},
			Refused: func(*inboundmessageservice.HandOffRequest) string {
				return "Would hand the Triage queue to one person."
			},
			Render: func(
				_ *inboundmessageservice.HandOffRequest,
				plan *inboundmessageservice.QueuePlan,
			) (*agent.ToolPreview, error) {
				count := countMessages(len(plan.Changes))
				if plan.More {
					count = "at least " + count
				}

				return renderQueuePlan("Would hand "+count+" waiting in Triage to one person",
					plan)
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.HandOffRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				result, err := queue.HandOff(ctx, req)
				if err != nil {
					return nil, err
				}

				return &agent.ToolExecutionResult{
					Action: "handed off",
					Kind:   "inbound messages",
					Name:   countMessages(result.Affected),
					IDs:    map[string]string{paramToUserID: idText(result.ToUserID)},
					Total:  result.Affected,
				}, nil
			},
		},
	)
}
