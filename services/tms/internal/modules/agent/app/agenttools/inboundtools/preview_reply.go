package inboundtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

// composedReply is a reply as it would go out: to the sender alone, in the
// sender's thread, under the original subject, in the organization's
// letterhead.
type composedReply struct {
	reply     *inboundReply
	data      documenttemplate.AgentEmailContext
	send      *emailports.SendEmailRequest
	versionID *pulid.ID
}

func (t *ReplyToInboundMessageTool) compose(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*composedReply, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	reply, err := t.prepare(ctx, *params)
	if err != nil {
		return nil, err
	}

	tenant := common.TenantFrom(*params)
	data := documenttemplate.AgentEmailContext{
		AgentSubject: reply.subject,
		AgentBody:    reply.body,
	}
	t.describeMatch(ctx, reply.message, tenant, &data)
	customerupdateservice.Brand(ctx, t.deps.orgRepo, t.deps.inliner, tenant, &data)

	rendered, err := t.deps.templates.RenderMessage(
		ctx,
		&documenttemplateports.RenderMessageRequest{
			TenantInfo: tenant,
			Kind:       documenttemplate.KindAgentInboundReplyEmail,
			CustomerID: pulid.PtrOrNil(reply.message.MatchedCustomerID),
			Data:       data,
		},
	)
	if err != nil {
		return nil, err
	}

	return &composedReply{
		reply:     reply,
		data:      data,
		versionID: rendered.VersionID,
		send: &emailports.SendEmailRequest{
			TenantInfo:     tenant,
			ProfileID:      reply.profileID,
			Purpose:        email.PurposeOperations,
			To:             []string{reply.message.FromAddress},
			Subject:        rendered.Subject,
			HTML:           rendered.HTML,
			Text:           rendered.Text,
			Headers:        threadHeaders(reply.message),
			IdempotencyKey: params.IdempotencyKey,
		},
	}, nil
}

// replyReviewNote is what the settled message records of the reply.
func replyReviewNote(reply *inboundReply) string {
	return stringutils.TruncateRunes(
		"Replied to "+reply.message.FromAddress+": "+reply.body,
		maxInboundReviewNote,
	)
}

func (t *ReplyToInboundMessageTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return nil, err
	}
	message := composed.reply.message

	sender, err := common.ResolveEmailSender(ctx, t.deps.senders, composed.send)
	if err != nil {
		return nil, err
	}

	send := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceInboundMessage,
			ID:       message.ID,
			Label:    message.FromAddress,
		},
		common.EmailMessagePreview(
			sender,
			common.RenderedEmailPreview(composed.send, composed.versionID),
		),
	)

	settled, err := toolpreview.Update(
		toolpreview.Record{
			Resource: permission.ResourceInboundMessage,
			ID:       message.ID,
			Label:    describeInboundMessage(message),
			Version:  common.PinnedVersion(message.Version),
		},
		message,
		func(after *inboundmessage.InboundMessage) error {
			after.ApplyReview(
				inboundmessage.StatusActioned,
				params.Actor.UserID,
				replyReviewNote(composed.reply),
				timeutils.NowUnix(),
			)

			return nil
		},
		toolpreview.Only("status", "reviewedBy", "reviewedAt", "reviewNote"),
		toolpreview.WithRefs(map[string]permission.Resource{"reviewedBy": permission.ResourceUser}),
		toolpreview.Volatile("reviewedAt"),
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(
		"Would reply to "+message.FromAddress+" in the thread "+
			describeInboundMessage(message)+", and mark the message actioned.",
		send,
		settled,
	)
	common.WarnSuppressedRecipients(preview, sender)
	common.WarnSensitiveContent(preview, composed.reply.body)

	return preview, nil
}
