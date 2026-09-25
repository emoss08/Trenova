package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ serviceports.ToolPreviewer = (*emailCustomerTool)(nil)
	_ serviceports.ToolPreviewer = (*requestMissingDocsTool)(nil)
	_ serviceports.ToolPreviewer = (*replyToInboundMessageTool)(nil)
)

// ------------------------------------------------------------ email_customer

// customerEmail is an update to a shipment's customer as it would go out:
// the recipients from the customer's own profile, the prose in the
// organization's letterhead, and the comment that records it. A customer
// already told within the hour is sent nothing, and nothing is rendered.
type customerEmail struct {
	shipment    *shipment.Shipment
	tenant      pagination.TenantInfo
	alreadyTold bool
	recipients  []string
	data        documenttemplate.AgentEmailContext
	send        *serviceports.SendEmailRequest
	versionID   *pulid.ID
}

func (t *emailCustomerTool) compose(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*customerEmail, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, previewFieldShipmentID)
	if err != nil {
		return nil, err
	}
	profileID, err := requirePulid(params.Params, "profileId")
	if err != nil {
		return nil, err
	}
	subject, err := requireString(params.Params, previewFieldSubject)
	if err != nil {
		return nil, err
	}
	body, err := requireString(params.Params, "body")
	if err != nil {
		return nil, err
	}

	tenant := tenantFrom(*params)
	sp, err := t.deps.shipments.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              shipmentID,
		TenantInfo:      tenant,
		ShipmentOptions: repositories.ShipmentOptions{IncludeCustomer: true},
	})
	if err != nil {
		return nil, err
	}

	composed := &customerEmail{shipment: sp, tenant: tenant}

	// The customer update desk is woken by every arrival and every
	// departure, so a shipment crossing a yard can raise several runs in a
	// few minutes. Telling the customer once is the rule; reading back what
	// was already sent is what enforces it.
	composed.alreadyTold, err = alreadyToldCustomer(
		ctx, t.deps.comments, tenant, sp.ID, timeutils.NowUnix(),
	)
	if err != nil || composed.alreadyTold {
		return composed, err
	}

	recipients, customerName, err := t.recipients(ctx, sp, tenant)
	if err != nil {
		return nil, err
	}
	composed.recipients = recipients

	composed.data = documenttemplate.AgentEmailContext{
		AgentSubject:      strings.TrimSpace(subject),
		AgentBody:         strings.TrimSpace(body),
		CustomerName:      customerName,
		ShipmentProNumber: sp.ProNumber,
	}
	brandAgentEmail(ctx, t.deps.orgRepo, t.deps.inliner, tenant, &composed.data)

	rendered, err := t.deps.templates.RenderMessage(ctx, &serviceports.RenderMessageRequest{
		TenantInfo: tenant,
		Kind:       documenttemplate.KindAgentCustomerUpdateEmail,
		CustomerID: pulid.PtrOrNil(sp.CustomerID),
		Data:       composed.data,
	})
	if err != nil {
		return nil, err
	}
	composed.versionID = rendered.VersionID

	composed.send = &serviceports.SendEmailRequest{
		TenantInfo:     tenant,
		ProfileID:      profileID,
		Purpose:        email.PurposeOperations,
		To:             recipients,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: params.IdempotencyKey,
	}

	return composed, nil
}

func (t *emailCustomerTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return nil, err
	}

	if composed.alreadyTold {
		preview := toolpreview.Build(fmt.Sprintf(
			"Would send nothing: the customer was already emailed about %s within the hour.",
			composed.shipment.ProNumber,
		))
		toolpreview.Warn(preview, agent.PreviewWarningAlreadyToldCustomer,
			"This customer was already emailed about this shipment within the hour.",
			composed.shipment.ProNumber)

		return preview, nil
	}

	sender, err := resolveEmailSender(ctx, t.deps.senders, composed.send)
	if err != nil {
		return nil, err
	}

	send := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceCustomer,
			ID:       composed.shipment.CustomerID,
			Label:    composed.data.CustomerName,
		},
		emailMessagePreview(sender, renderedEmailPreview(composed.send, composed.versionID)),
	)

	comment := customerUpdateComment(composed)
	recorded, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceShipmentComment,
			Label:    "Comment on " + composed.shipment.ProNumber,
		},
		&shipment.ShipmentComment{
			Comment:    comment.Comment,
			Type:       comment.Type,
			Visibility: comment.Visibility,
		},
		toolpreview.Only("comment", "type", "visibility"),
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would email %s about %s, to %s, and record it on the shipment.",
			composed.data.CustomerName,
			composed.shipment.ProNumber,
			strings.Join(composed.recipients, ", ")),
		send,
		recorded,
	)
	warnSuppressedRecipients(preview, sender)
	warnSensitiveContent(preview, composed.data.AgentSubject, composed.data.AgentBody)

	return preview, nil
}

// customerUpdateComment is what the shipment's thread records of an email to
// its customer: who it went to and what it said, stamped as the agent's.
func customerUpdateComment(
	composed *customerEmail,
) *serviceports.CreateSystemShipmentCommentRequest {
	return &serviceports.CreateSystemShipmentCommentRequest{
		TenantInfo: composed.tenant,
		ShipmentID: composed.shipment.ID,
		Comment: fmt.Sprintf(
			"Emailed %s: %s\n\n%s",
			strings.Join(composed.recipients, ", "),
			composed.data.AgentSubject,
			composed.data.AgentBody,
		),
		Type:       shipment.CommentTypeCustomerUpdate,
		Visibility: shipment.CommentVisibilityOperations,
		Priority:   shipment.CommentPriorityNormal,
		Metadata: map[string]any{
			shipment.CommentMetadataOrigin: shipment.CommentOriginAgent,
			"tool":                         emailCustomerToolName,
			"recipients":                   composed.recipients,
			previewFieldSubject:            composed.data.AgentSubject,
		},
	}
}

// ------------------------------------------------------ request_missing_docs

// docsRequest is a request for paperwork as it would go out: to the
// addresses the call names, in the organization's letterhead, with no
// fallback when the template fails.
type docsRequest struct {
	data      documenttemplate.AgentEmailContext
	send      *serviceports.SendEmailRequest
	versionID *pulid.ID
}

func (t *requestMissingDocsTool) compose(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*docsRequest, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	profileID, err := requirePulid(params.Params, "profileId")
	if err != nil {
		return nil, err
	}
	subject, err := requireString(params.Params, previewFieldSubject)
	if err != nil {
		return nil, err
	}
	body, err := requireString(params.Params, "body")
	if err != nil {
		return nil, err
	}

	var to []string
	if err = decodeParam(params.Params, "to", &to); err != nil {
		return nil, err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: params.OrganizationID,
		BuID:  params.BusinessUnitID,
	}

	// The agent's prose is wrapped by the organization's template rather than
	// being the whole message, and it renders with no fallback: a template a
	// carrier authored badly must not be silently replaced on a message sent
	// under their letterhead.
	data := t.agentEmailContext(ctx, tenantInfo, subject, body, params.Params)
	rendered, err := t.templates.RenderMessage(ctx, &serviceports.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindAgentRequestMissingDocsEmail,
		Data:       data,
	})
	if err != nil {
		return nil, err
	}

	return &docsRequest{
		data:      data,
		versionID: rendered.VersionID,
		send: &serviceports.SendEmailRequest{
			TenantInfo:     tenantInfo,
			ProfileID:      profileID,
			Purpose:        email.PurposeBilling,
			To:             to,
			Subject:        rendered.Subject,
			HTML:           rendered.HTML,
			Text:           rendered.Text,
			IdempotencyKey: params.IdempotencyKey,
		},
	}, nil
}

func (t *requestMissingDocsTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return nil, err
	}

	sender, err := resolveEmailSender(ctx, t.senders, composed.send)
	if err != nil {
		return nil, err
	}

	label := stringutils.FirstNonEmpty(
		composed.data.CustomerName,
		strings.Join(composed.send.To, ", "),
	)
	send := toolpreview.Send(
		toolpreview.Record{Resource: permission.ResourceCustomerCommunication, Label: label},
		emailMessagePreview(sender, renderedEmailPreview(composed.send, composed.versionID)),
	)

	summary := "Would email " + strings.Join(composed.send.To, ", ") +
		" asking for the missing paperwork"
	if len(composed.data.RequestedDocuments) > 0 {
		summary += ": " + strings.Join(composed.data.RequestedDocuments, ", ")
	}
	if composed.data.ShipmentProNumber != "" {
		summary += " for " + composed.data.ShipmentProNumber
	}

	preview := toolpreview.Build(summary+". It cannot be recalled once sent.", send)
	warnSuppressedRecipients(preview, sender)
	warnSensitiveContent(preview, composed.data.AgentSubject, composed.data.AgentBody)

	return preview, nil
}

// --------------------------------------------------- reply_to_inbound_message

// composedReply is a reply as it would go out: to the sender alone, in the
// sender's thread, under the original subject, in the organization's
// letterhead.
type composedReply struct {
	reply     *inboundReply
	data      documenttemplate.AgentEmailContext
	send      *serviceports.SendEmailRequest
	versionID *pulid.ID
}

func (t *replyToInboundMessageTool) compose(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*composedReply, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	reply, err := t.prepare(ctx, *params)
	if err != nil {
		return nil, err
	}

	tenant := tenantFrom(*params)
	data := documenttemplate.AgentEmailContext{
		AgentSubject: reply.subject,
		AgentBody:    reply.body,
	}
	t.describeMatch(ctx, reply.message, tenant, &data)
	brandAgentEmail(ctx, t.deps.orgRepo, t.deps.inliner, tenant, &data)

	rendered, err := t.deps.templates.RenderMessage(ctx, &serviceports.RenderMessageRequest{
		TenantInfo: tenant,
		Kind:       documenttemplate.KindAgentInboundReplyEmail,
		CustomerID: pulid.PtrOrNil(reply.message.MatchedCustomerID),
		Data:       data,
	})
	if err != nil {
		return nil, err
	}

	return &composedReply{
		reply:     reply,
		data:      data,
		versionID: rendered.VersionID,
		send: &serviceports.SendEmailRequest{
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

func (t *replyToInboundMessageTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return nil, err
	}
	message := composed.reply.message

	sender, err := resolveEmailSender(ctx, t.deps.senders, composed.send)
	if err != nil {
		return nil, err
	}

	send := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceInboundMessage,
			ID:       message.ID,
			Label:    message.FromAddress,
		},
		emailMessagePreview(sender, renderedEmailPreview(composed.send, composed.versionID)),
	)

	settled, err := toolpreview.Update(
		toolpreview.Record{
			Resource: permission.ResourceInboundMessage,
			ID:       message.ID,
			Label:    describeInboundMessage(message),
			Version:  pinnedVersion(message.Version),
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
	warnSuppressedRecipients(preview, sender)
	warnSensitiveContent(preview, composed.reply.body)

	return preview, nil
}

// renderedEmailPreview is the message a send request would deliver: the
// rendered subject, the plain-text body a reader sees, and its recipients.
func renderedEmailPreview(
	send *serviceports.SendEmailRequest,
	versionID *pulid.ID,
) *agent.MessagePreview {
	message := &agent.MessagePreview{
		To:      send.To,
		Cc:      send.CC,
		Bcc:     send.BCC,
		Subject: send.Subject,
		Body:    send.Text,
	}
	for _, attachment := range send.Attachments {
		message.Attachments = append(message.Attachments, attachment.FileName)
	}
	if versionID != nil {
		message.TemplateVersionID = *versionID
	}

	return message
}
