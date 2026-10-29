package triagequeuetools

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/inboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
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
	toolpreview.Only(changepreview.FieldStatus, "reviewNote", "reviewedBy", "reviewedAt", "dueAt"),
	toolpreview.WithRefs(map[string]permission.Resource{"reviewedBy": permission.ResourceUser}),
	toolpreview.Volatile("reviewedAt"),
}

type blockedSendersView struct {
	BlockedSenders []string `json:"blockedSenders"`
}

func mailboxSendersChange(plan *triageworkservice.SenderPlan) (*agentpreview.RecordChange, error) {
	return toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceInboundMailbox,
			ID:       plan.Mailbox.ID,
			Label:    plan.Mailbox.Address,
			Version:  changepreview.PreviewVersion(plan.Mailbox.Version),
		},
		&blockedSendersView{BlockedSenders: plan.Mailbox.BlockedSenders},
		&blockedSendersView{BlockedSenders: plan.Blocked},
		toolpreview.Labels(map[string]string{"blockedSenders": "Blocked senders"}),
	)
}

func NewBlockInboundSenderTool(p inboundtools.TriageToolParams) agenttoolports.AgentTool {
	return newBlockInboundSenderTool(p.Work)
}

func newBlockInboundSenderTool(senders triageSenders) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:     "inbox_block_sender",
			Artifact: inboundtools.MessageEntity,
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
				inboundtools.ParamMessageID: inboundtools.MessageIDProperty(
					"A message from the sender, from inbox_list_inbound_messages, " +
						"newest first or by query.",
				),
			},
			Required:    []string{inboundtools.ParamMessageID},
			SearchTerms: []string{"block sender", "phishing", "junk sender"},
			Recipe:      []string{inboundtools.ToolGetMessage, "inbox_block_sender"},
			Target:      inboundtools.TargetMessage,
		},
		toolspec.ReceivablePlan[*triageworkservice.BlockSenderRequest, *triageworkservice.SenderPlan]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*triageworkservice.BlockSenderRequest, error) {
				messageID, err := toolparams.RequirePulid(
					params.Params,
					inboundtools.ParamMessageID,
				)
				if err != nil {
					return nil, err
				}

				return &triageworkservice.BlockSenderRequest{
					TenantInfo: inboundtools.TenantOnly(params),
					UserID:     inboundtools.ActorOf(params),
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
			) (*agentpreview.ToolPreview, error) {
				mailbox, err := mailboxSendersChange(plan)
				if err != nil {
					return nil, err
				}
				settled, err := inboundtools.MessageChange(plan.Message, plan.After,
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
			) (*agentrecord.ToolExecutionResult, error) {
				message, err := senders.BlockSender(ctx, req)
				if err != nil {
					return nil, err
				}

				return inboundtools.MessageResult("blocked the sender of", message), nil
			},
		},
	)
}

func NewUnblockInboundSenderTool(p inboundtools.TriageToolParams) agenttoolports.AgentTool {
	return newUnblockInboundSenderTool(p.Inbox, p.Work)
}

func newUnblockInboundSenderTool(
	inbox inboundtools.MessageReader,
	senders triageSenders,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:     "inbox_unblock_sender",
			Artifact: inboundtools.MessageEntity,
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
				inboundtools.ParamMessageID: inboundtools.MessageIDProperty(
					"A message from the blocked sender, from inbox_list_inbound_messages " +
						"with a query naming the sender.",
				),
			},
			Required:    []string{inboundtools.ParamMessageID},
			SearchTerms: []string{"unblock sender", "allow sender"},
			Recipe:      []string{"inbox_list_inbound_messages", "inbox_unblock_sender"},
			Target:      inboundtools.TargetMessage,
		},
		toolspec.ReceivablePlan[
			*inboundqueue.MessageActionRequest,
			*triageworkservice.SenderPlan,
		]{
			Request: inboundtools.MessageAction,
			Plan: func(
				ctx context.Context,
				req *inboundqueue.MessageActionRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*triageworkservice.SenderPlan, error) {
				unblock, err := unblockRequest(ctx, inbox, req)
				if err != nil {
					return nil, err
				}

				return senders.PlanUnblockSender(ctx, unblock)
			},
			Refused: func(*inboundqueue.MessageActionRequest) string {
				return "Would let a blocked sender's mail through again."
			},
			Render: renderUnblockSender,
			Run: func(
				ctx context.Context,
				req *inboundqueue.MessageActionRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
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

				return &agentrecord.ToolExecutionResult{
					Action: "unblocked",
					Kind:   "sender",
					Name:   unblock.Sender + " on " + mailbox.Address,
					IDs:    map[string]string{inboundtools.ParamMessageID: req.MessageID.String()},
				}, nil
			},
		},
	)
}

func unblockRequest(
	ctx context.Context,
	inbox inboundtools.MessageReader,
	req *inboundqueue.MessageActionRequest,
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
	_ *inboundqueue.MessageActionRequest,
	plan *triageworkservice.SenderPlan,
) (*agentpreview.ToolPreview, error) {
	summary := "Would let mail from " + plan.Sender + " on " + plan.Mailbox.Address +
		" reach the Triage queue again."
	if !plan.Changed {
		return inboundtools.RefusedPreview(summary, errSenderNotBlocked), nil
	}
	mailbox, err := mailboxSendersChange(plan)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, mailbox), nil
}
