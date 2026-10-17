package inboundtools

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramReplyBody    = "body"
	paramReplySubject = "subject"
	maxReplySubject   = 500
)

type triageReplies interface {
	DraftReply(
		ctx context.Context,
		req *inboundreplyservice.DraftReplyRequest,
	) (*triage.Reply, error)
	PlanDraftReply(
		ctx context.Context,
		req *inboundreplyservice.DraftReplyRequest,
	) (*inboundreplyservice.DraftPlan, error)
	SaveDraft(
		ctx context.Context,
		req *inboundreplyservice.SaveDraftRequest,
	) (*triage.Reply, error)
	PlanSaveDraft(
		ctx context.Context,
		req *inboundreplyservice.SaveDraftRequest,
	) (*inboundreplyservice.DraftPlan, error)
	DiscardDraft(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		messageID pulid.ID,
	) (*inboundmessage.InboundMessage, error)
	PlanDiscardDraft(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		messageID pulid.ID,
	) (*inboundreplyservice.DraftPlan, error)
}

var errNoReplyDraft = errors.New("this message has no reply draft to throw away")

type replyDraftView struct {
	ToAddress     string `json:"toAddress"`
	Subject       string `json:"subject"`
	Body          string `json:"body"`
	DraftedByNova bool   `json:"draftedByNova"`
}

var replyDraftLabels = map[string]string{
	"toAddress":     "To",
	"subject":       "Subject",
	paramReplyBody:  "Body",
	"draftedByNova": "Written by Nova",
}

func replyDraftViewOf(reply *triage.Reply) *replyDraftView {
	if reply == nil {
		return &replyDraftView{}
	}

	return &replyDraftView{
		ToAddress:     reply.ToAddress,
		Subject:       reply.Subject,
		Body:          reply.Body,
		DraftedByNova: reply.DraftedByNova,
	}
}

func replyDraftRecord(message *inboundmessage.InboundMessage) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceInboundMessage,
		ID:       message.ID,
		Label:    "Reply draft to " + describeInboundMessage(message),
	}
}

type draftReplyCall struct {
	save  *inboundreplyservice.SaveDraftRequest
	write *inboundreplyservice.DraftReplyRequest
}

func NewDraftInboundReplyTool(p triageToolParams) agentports.AgentTool {
	return newDraftInboundReplyTool(p.Inbox, p.Replies)
}

func newDraftInboundReplyTool(
	inbox inboundMessageReader,
	replies triageReplies,
) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "draft_inbound_reply",
			Artifact: inboundMessageEntity,
			Description: "Save a reply to a message as its draft in Triage, for a person to " +
				"review and send; nothing is sent. Write the body yourself, or leave it out to " +
				"have Triage's drafter write it from the message, its linked records and, for " +
				"a detention dispute, the stop times. It replaces any draft already there. " +
				"To answer the sender yourself, use reply_to_inbound_message instead.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundCondition(inbox),
			Reversible:  true,
			Rationale: "Writes a draft only people inside the organization see; sending it " +
				"stays with a person.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The message to answer, from list_inbound_messages or search_inbound_messages.",
				),
				paramReplyBody: common.StringProperty(
					"The reply in plain prose, as the person should send it. Leave it out to "+
						"have Triage's drafter write it.",
					maxInboundReplyBody,
				),
				paramReplySubject: common.StringProperty(
					"Optional, with a body only: the subject. Defaults to Re: and the "+
						"message's own subject.",
					maxReplySubject,
				),
			},
			Required:    []string{paramMessageID},
			SearchTerms: []string{"draft a reply", "prepare an answer", "write a response"},
			Recipe:      []string{toolGetInboundMessage, "draft_inbound_reply"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[*draftReplyCall, *inboundreplyservice.DraftPlan]{
			Request: draftReplyRequest,
			Plan: func(
				ctx context.Context,
				call *draftReplyCall,
				_ *agentports.ToolExecuteParams,
			) (*inboundreplyservice.DraftPlan, error) {
				if call.save != nil {
					return replies.PlanSaveDraft(ctx, call.save)
				}

				return replies.PlanDraftReply(ctx, call.write)
			},
			Refused: func(*draftReplyCall) string {
				return "Would save a reply draft on an inbound message."
			},
			Render: renderReplyDraft,
			Run: func(
				ctx context.Context,
				call *draftReplyCall,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				var (
					reply *triage.Reply
					err   error
				)
				if call.save != nil {
					reply, err = replies.SaveDraft(ctx, call.save)
				} else {
					reply, err = replies.DraftReply(ctx, call.write)
				}
				if err != nil {
					return nil, err
				}

				return &agent.ToolExecutionResult{
					Action: "drafted",
					Kind:   "reply draft",
					Name:   reply.Subject,
					IDs:    map[string]string{paramMessageID: reply.MessageID.String()},
					Record: common.RecordOf(inboundMessageEntity, reply.MessageID),
				}, nil
			},
		},
	)
}

func draftReplyRequest(params *agentports.ToolExecuteParams) (*draftReplyCall, error) {
	messageID, err := common.RequirePulid(params.Params, paramMessageID)
	if err != nil {
		return nil, err
	}
	body, err := common.BoundedText(params.Params, paramReplyBody, maxInboundReplyBody)
	if err != nil {
		return nil, err
	}
	subject, err := common.BoundedText(params.Params, paramReplySubject, maxReplySubject)
	if err != nil {
		return nil, err
	}

	if body == "" {
		if subject != "" {
			return nil, errors.New(`"subject" is only taken with a "body"`)
		}

		return &draftReplyCall{write: &inboundreplyservice.DraftReplyRequest{
			TenantInfo: tenantOnly(params),
			UserID:     actorOf(params),
			MessageID:  messageID,
			Mode:       triagenova.Interactive,
		}}, nil
	}

	return &draftReplyCall{save: &inboundreplyservice.SaveDraftRequest{
		TenantInfo: tenantOnly(params),
		UserID:     actorOf(params),
		MessageID:  messageID,
		Subject:    subject,
		Body:       body,
	}}, nil
}

func renderReplyDraft(
	call *draftReplyCall,
	plan *inboundreplyservice.DraftPlan,
) (*agent.ToolPreview, error) {
	opts := []toolpreview.Option{toolpreview.Labels(replyDraftLabels)}
	summary := "Would save the reply as the message's draft for a person to review and send"
	if call.write != nil {
		opts = append(opts, toolpreview.Ignore("body"))
		summary = "Would have Triage's drafter write a reply and save it as the message's " +
			"draft for a person to review and send; the body is written when this runs"
	}
	if plan.Current != nil {
		summary += ", replacing the draft already there"
	}

	var (
		change *agent.RecordChange
		err    error
	)
	if plan.Current == nil {
		change, err = toolpreview.Create(replyDraftRecord(plan.Message),
			replyDraftViewOf(plan.Draft), opts...)
	} else {
		change, err = toolpreview.Changed(replyDraftRecord(plan.Message),
			replyDraftViewOf(plan.Current), replyDraftViewOf(plan.Draft), opts...)
	}
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(summary+". Nothing is sent.", change)
	preview.Partial = call.write != nil

	return preview, nil
}

func NewDiscardInboundReplyDraftTool(p triageToolParams) agentports.AgentTool {
	return newDiscardInboundReplyDraftTool(p.Replies)
}

func newDiscardInboundReplyDraftTool(replies triageReplies) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "discard_inbound_reply_draft",
			Artifact: inboundMessageEntity,
			Description: "Throw away the reply draft waiting on a message in Triage, when it " +
				"should not be sent at all. get_inbound_message shows whether a message has " +
				"one. A person's own draft is lost with it; to change a draft, write a new " +
				"one with draft_inbound_reply instead.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Rationale: "Deletes a draft that may be a person's own unsent words, with nothing " +
				"to bring it back.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The message, from list_inbound_messages or search_inbound_messages.",
				),
			},
			Required:    []string{paramMessageID},
			SearchTerms: []string{"discard draft", "delete reply draft", "throw away reply"},
			Recipe:      []string{toolGetInboundMessage, "discard_inbound_reply_draft"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[
			*inboundmessageservice.MessageActionRequest,
			*inboundreplyservice.DraftPlan,
		]{
			Request: messageAction,
			Plan: func(
				ctx context.Context,
				req *inboundmessageservice.MessageActionRequest,
				_ *agentports.ToolExecuteParams,
			) (*inboundreplyservice.DraftPlan, error) {
				return replies.PlanDiscardDraft(ctx, req.TenantInfo, req.MessageID)
			},
			Refused: func(*inboundmessageservice.MessageActionRequest) string {
				return "Would throw away a reply draft."
			},
			Render: func(
				_ *inboundmessageservice.MessageActionRequest,
				plan *inboundreplyservice.DraftPlan,
			) (*agent.ToolPreview, error) {
				summary := "Would throw away the reply draft on " +
					describeInboundMessage(plan.Message) + "."
				if plan.Current == nil {
					return refusedPreview(summary, errNoReplyDraft), nil
				}
				change, err := toolpreview.Delete(replyDraftRecord(plan.Message),
					replyDraftViewOf(plan.Current), toolpreview.Labels(replyDraftLabels))
				if err != nil {
					return nil, err
				}

				return toolpreview.Build(summary, change), nil
			},
			Run: func(
				ctx context.Context,
				req *inboundmessageservice.MessageActionRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				plan, err := replies.PlanDiscardDraft(ctx, req.TenantInfo, req.MessageID)
				if err != nil {
					return nil, err
				}
				if plan.Current == nil {
					return nil, errNoReplyDraft
				}
				message, err := replies.DiscardDraft(ctx, req.TenantInfo, req.MessageID)
				if err != nil {
					return nil, err
				}

				return messageResult("discarded the reply draft of", message), nil
			},
		},
	)
}
