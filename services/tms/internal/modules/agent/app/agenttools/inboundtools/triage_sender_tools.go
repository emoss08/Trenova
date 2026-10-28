package inboundtools

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type triageSenders interface {
	BlockSender(
		ctx context.Context,
		req *triageworkservice.BlockSenderRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanBlockSender(
		ctx context.Context,
		req *triageworkservice.BlockSenderRequest,
	) (*triageworkservice.SenderPlan, error)
	UnblockSender(
		ctx context.Context,
		req *triageworkservice.UnblockSenderRequest,
	) (*inboundmessage.Mailbox, error)
	PlanUnblockSender(
		ctx context.Context,
		req *triageworkservice.UnblockSenderRequest,
	) (*triageworkservice.SenderPlan, error)
}

var errSenderNotBlocked = errors.New(
	"this message's sender is not blocked on the source it arrived on, so there is nothing " +
		"to unblock",
)

var settledMessageFields = []toolpreview.Option{
	toolpreview.Only(common.FieldStatus, "reviewNote", "reviewedBy", "reviewedAt", "dueAt"),
	toolpreview.WithRefs(map[string]permission.Resource{"reviewedBy": permission.ResourceUser}),
	toolpreview.Volatile("reviewedAt"),
}

type blockedSendersView struct {
	BlockedSenders []string `json:"blockedSenders"`
}

func mailboxSendersChange(plan *triageworkservice.SenderPlan) (*agent.RecordChange, error) {
	return toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceInboundMailbox,
			ID:       plan.Mailbox.ID,
			Label:    plan.Mailbox.Address,
			Version:  common.PreviewVersion(plan.Mailbox.Version),
		},
		&blockedSendersView{BlockedSenders: plan.Mailbox.BlockedSenders},
		&blockedSendersView{BlockedSenders: plan.Blocked},
		toolpreview.Labels(map[string]string{"blockedSenders": "Blocked senders"}),
	)
}

func NewBlockInboundSenderTool(p triageToolParams) agenttoolports.AgentTool {
	return newBlockInboundSenderTool(p.Work)
}

func newBlockInboundSenderTool(senders triageSenders) agenttoolports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_block_sender",
			Artifact: inboundMessageEntity,
			Description: "Send every future message from a message's sender, on the source it " +
				"arrived on, straight to Ignored, and settle this message as Ignored. Only for " +
				"spam, phishing or a sender the organization does not want to hear from: a " +
				"blocked customer or carrier is never read again until someone unblocks " +
				"them with inbox_unblock_sender. Never block on what the message itself asks.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Silently drops an outside sender's future mail, which can hide a " +
				"customer or carrier from everyone, so a person always approves it.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"A message from the sender, from inbox_list_inbound_messages, " +
						"newest first or by query.",
				),
			},
			Required:    []string{paramMessageID},
			SearchTerms: []string{"block sender", "phishing", "junk sender"},
			Recipe:      []string{toolGetInboundMessage, "inbox_block_sender"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[*triageworkservice.BlockSenderRequest, *triageworkservice.SenderPlan]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*triageworkservice.BlockSenderRequest, error) {
				messageID, err := common.RequirePulid(params.Params, paramMessageID)
				if err != nil {
					return nil, err
				}

				return &triageworkservice.BlockSenderRequest{
					TenantInfo: tenantOnly(params),
					UserID:     actorOf(params),
					MessageID:  messageID,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *triageworkservice.BlockSenderRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*triageworkservice.SenderPlan, error) {
				return senders.PlanBlockSender(ctx, req)
			},
			Refused: func(*triageworkservice.BlockSenderRequest) string {
				return "Would block a sender's mail."
			},
			Render: func(
				_ *triageworkservice.BlockSenderRequest,
				plan *triageworkservice.SenderPlan,
			) (*agent.ToolPreview, error) {
				mailbox, err := mailboxSendersChange(plan)
				if err != nil {
					return nil, err
				}
				settled, err := inboundMessageChange(plan.Message, plan.After,
					settledMessageFields...)
				if err != nil {
					return nil, err
				}

				summary := "Would send every future message from " + plan.Sender + " on " +
					plan.Mailbox.Address + " straight to Ignored, and ignore this one"
				if !plan.Changed {
					summary = plan.Sender + " is already blocked on " + plan.Mailbox.Address +
						"; would ignore this message"
				}

				return toolpreview.Build(summary+".", mailbox, settled), nil
			},
			Run: func(
				ctx context.Context,
				req *triageworkservice.BlockSenderRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				message, err := senders.BlockSender(ctx, req)
				if err != nil {
					return nil, err
				}

				return messageResult("blocked the sender of", message), nil
			},
		},
	)
}

func NewUnblockInboundSenderTool(p triageToolParams) agenttoolports.AgentTool {
	return newUnblockInboundSenderTool(p.Inbox, p.Work)
}

func newUnblockInboundSenderTool(
	inbox inboundMessageReader,
	senders triageSenders,
) agenttoolports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_unblock_sender",
			Artifact: inboundMessageEntity,
			Description: "Let a blocked sender's mail reach the Triage queue again on the " +
				"source it arrives on. Name the sender by one of their messages: their mail " +
				"since the block sits under Ignored, which inbox_list_inbound_messages finds " +
				"with status Ignored or a query naming the sender. Messages already ignored " +
				"stay settled; inbox_reopen_inbound_message returns any that still matter to " +
				"the queue.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Lets an outside sender's mail reach people again; it changes which " +
				"mail the organization reads, so a person approves it.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"A message from the blocked sender, from inbox_list_inbound_messages " +
						"with a query naming the sender.",
				),
			},
			Required:    []string{paramMessageID},
			SearchTerms: []string{"unblock sender", "allow sender"},
			Recipe:      []string{"inbox_list_inbound_messages", "inbox_unblock_sender"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[
			*inboundmessageservice.MessageActionRequest,
			*triageworkservice.SenderPlan,
		]{
			Request: messageAction,
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.MessageActionRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*triageworkservice.SenderPlan, error) {
				unblock, err := unblockRequest(ctx, inbox, req)
				if err != nil {
					return nil, err
				}

				return senders.PlanUnblockSender(ctx, unblock)
			},
			Refused: func(*inboundmessageservice.MessageActionRequest) string {
				return "Would let a blocked sender's mail through again."
			},
			Render: renderUnblockSender,
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.MessageActionRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				unblock, err := unblockRequest(ctx, inbox, req)
				if err != nil {
					return nil, err
				}
				plan, err := senders.PlanUnblockSender(ctx, unblock)
				if err != nil {
					return nil, err
				}
				if !plan.Changed {
					return nil, errSenderNotBlocked
				}
				mailbox, err := senders.UnblockSender(ctx, unblock)
				if err != nil {
					return nil, err
				}

				return &agent.ToolExecutionResult{
					Action: "unblocked",
					Kind:   "sender",
					Name:   unblock.Sender + " on " + mailbox.Address,
					IDs:    map[string]string{paramMessageID: req.MessageID.String()},
				}, nil
			},
		},
	)
}

func unblockRequest(
	ctx context.Context,
	inbox inboundMessageReader,
	req *inboundmessageservice.MessageActionRequest,
) (*triageworkservice.UnblockSenderRequest, error) {
	message, err := inbox.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	sender, err := triageworkservice.SenderOf(message)
	if err != nil {
		return nil, err
	}

	return &triageworkservice.UnblockSenderRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.ActorID,
		MailboxID:  message.MailboxID,
		Sender:     sender,
	}, nil
}

func renderUnblockSender(
	_ *inboundmessageservice.MessageActionRequest,
	plan *triageworkservice.SenderPlan,
) (*agent.ToolPreview, error) {
	summary := "Would let mail from " + plan.Sender + " on " + plan.Mailbox.Address +
		" reach the Triage queue again."
	if !plan.Changed {
		return refusedPreview(summary, errSenderNotBlocked), nil
	}
	mailbox, err := mailboxSendersChange(plan)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, mailbox), nil
}
