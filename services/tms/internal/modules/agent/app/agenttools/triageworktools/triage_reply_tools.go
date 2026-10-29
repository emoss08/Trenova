package triageworktools

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
		Label:    "Reply draft to " + inboundtools.DescribeMessage(message),
	}
}

type draftReplyCall struct {
	save  *inboundreplyservice.SaveDraftRequest
	write *inboundreplyservice.DraftReplyRequest
}

func NewDraftInboundReplyTool(p inboundtools.TriageToolParams) agenttoolports.AgentTool {
	return newDraftInboundReplyTool(p.Inbox, p.Replies)
}

func newDraftInboundReplyTool(
	inbox inboundtools.MessageReader,
	replies triageReplies,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:     "inbox_draft_reply",
			Artifact: inboundtools.MessageEntity,
			Description: "Save a reply to a message as its draft in Triage, for a person to " +
				"review and send; nothing is sent. Write the body yourself, or leave it out to " +
				"have Triage's drafter write it from the message, its linked records and, for " +
				"a detention dispute, the stop times. It replaces any draft already there. " +
				"To answer the sender yourself, use inbox_send_reply instead.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundtools.MessageCondition(inbox),
			Reversible:  true,
			Rationale: "Writes a draft only people inside the organization see; sending it " +
				"stays with a person.",
			Properties: map[string]any{
				inboundtools.ParamMessageID: inboundtools.MessageIDProperty(
					"The message to answer, from inbox_list_inbound_messages, newest first or by query.",
				),
				paramReplyBody: toolparams.StringProperty(
					"The reply in plain prose, as the person should send it. Leave it out to "+
						"have Triage's drafter write it.",
					inboundtools.MaxReplyBody,
				),
				paramReplySubject: toolparams.StringProperty(
					"Optional, with a body only: the subject. Defaults to Re: and the "+
						"message's own subject.",
					maxReplySubject,
				),
			},
			Required:    []string{inboundtools.ParamMessageID},
			SearchTerms: []string{"draft a reply", "prepare an answer", "write a response"},
			Recipe:      []string{inboundtools.ToolGetMessage, "inbox_draft_reply"},
			Target:      inboundtools.TargetMessage,
		},
		toolspec.ReceivablePlan[*draftReplyCall, *inboundreplyservice.DraftPlan]{
			Request: draftReplyRequest,
			Plan: func(
				ctx context.Context,
				call *draftReplyCall,
				_ *agenttoolports.ToolExecuteParams,
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
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
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

				return &agentrecord.ToolExecutionResult{
					Action: "drafted",
					Kind:   "reply draft",
					Name:   reply.Subject,
					IDs: map[string]string{
						inboundtools.ParamMessageID: reply.MessageID.String(),
					},
					Record: changepreview.RecordOf(inboundtools.MessageEntity, reply.MessageID),
				}, nil
			},
		},
	)
}

func draftReplyRequest(params *agenttoolports.ToolExecuteParams) (*draftReplyCall, error) {
	messageID, err := toolparams.RequirePulid(params.Params, inboundtools.ParamMessageID)
	if err != nil {
		return nil, err
	}
	body, err := toolparams.BoundedText(params.Params, paramReplyBody, inboundtools.MaxReplyBody)
	if err != nil {
		return nil, err
	}
	subject, err := toolparams.BoundedText(params.Params, paramReplySubject, maxReplySubject)
	if err != nil {
		return nil, err
	}

	if body == "" {
		if subject != "" {
			return nil, errors.New(`"subject" is only taken with a "body"`)
		}

		return &draftReplyCall{write: &inboundreplyservice.DraftReplyRequest{
			TenantInfo: inboundtools.TenantOnly(params),
			UserID:     inboundtools.ActorOf(params),
			MessageID:  messageID,
			Mode:       triagenova.Interactive,
		}}, nil
	}

	return &draftReplyCall{save: &inboundreplyservice.SaveDraftRequest{
		TenantInfo: inboundtools.TenantOnly(params),
		UserID:     inboundtools.ActorOf(params),
		MessageID:  messageID,
		Subject:    subject,
		Body:       body,
	}}, nil
}

func renderReplyDraft(
	call *draftReplyCall,
	plan *inboundreplyservice.DraftPlan,
) (*agentpreview.ToolPreview, error) {
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
		change *agentpreview.RecordChange
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

func NewDiscardInboundReplyDraftTool(p inboundtools.TriageToolParams) agenttoolports.AgentTool {
	return newDiscardInboundReplyDraftTool(p.Replies)
}

func newDiscardInboundReplyDraftTool(replies triageReplies) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:     "inbox_discard_reply_draft",
			Artifact: inboundtools.MessageEntity,
			Description: "Throw away the reply draft waiting on a message in Triage, when it " +
				"should not be sent at all. inbox_get_inbound_message shows whether a message has " +
				"one. A person's own draft is lost with it; to change a draft, write a new " +
				"one with inbox_draft_reply instead.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Rationale: "Deletes a draft that may be a person's own unsent words, with nothing " +
				"to bring it back.",
			Properties: map[string]any{
				inboundtools.ParamMessageID: inboundtools.MessageIDProperty(
					"The message, from inbox_list_inbound_messages, newest first or by query.",
				),
			},
			Required:    []string{inboundtools.ParamMessageID},
			SearchTerms: []string{"discard draft", "delete reply draft", "throw away reply"},
			Recipe:      []string{inboundtools.ToolGetMessage, "inbox_discard_reply_draft"},
			Target:      inboundtools.TargetMessage,
		},
		toolspec.ReceivablePlan[
			*inboundqueue.MessageActionRequest,
			*inboundreplyservice.DraftPlan,
		]{
			Request: inboundtools.MessageAction,
			Plan: func(
				ctx context.Context,
				req *inboundqueue.MessageActionRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*inboundreplyservice.DraftPlan, error) {
				return replies.PlanDiscardDraft(ctx, req.TenantInfo, req.MessageID)
			},
			Refused: func(*inboundqueue.MessageActionRequest) string {
				return "Would throw away a reply draft."
			},
			Render: func(
				_ *inboundqueue.MessageActionRequest,
				plan *inboundreplyservice.DraftPlan,
			) (*agentpreview.ToolPreview, error) {
				summary := "Would throw away the reply draft on " +
					inboundtools.DescribeMessage(plan.Message) + "."
				if plan.Current == nil {
					return inboundtools.RefusedPreview(summary, errNoReplyDraft), nil
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
				req *inboundqueue.MessageActionRequest,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
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

				return inboundtools.MessageResult("discarded the reply draft of", message), nil
			},
		},
	)
}
