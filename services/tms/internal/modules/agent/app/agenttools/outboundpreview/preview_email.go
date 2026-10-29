package outboundpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// customerEmail is an update to a shipment's customer as it would go out:
// the recipients from the customer's own profile, the prose in the
// organization's letterhead, and the comment that records it. A customer
// already told within the hour is sent nothing, and nothing is rendered.
type CustomerEmail struct {
	Shipment    *shipment.Shipment                 `json:"-"`
	Tenant      pagination.TenantInfo              `json:"-"`
	AlreadyTold bool                               `json:"-"`
	Recipients  []string                           `json:"-"`
	Data        documenttemplate.AgentEmailContext `json:"-"`
	Send        *emailports.SendEmailRequest       `json:"-"`
	VersionID   *pulid.ID                          `json:"-"`
}

// customerUpdateComment is what the shipment's thread records of an email to
// its customer: who it went to and what it said, stamped as the agent's.
func CustomerUpdateComment(
	composed *CustomerEmail,
) *shipmentcommentports.CreateSystemShipmentCommentRequest {
	return customerupdateservice.Comment(&customerupdateservice.CommentParams{
		Tenant:     composed.Tenant,
		ShipmentID: composed.Shipment.ID,
		Recipients: composed.Recipients,
		Subject:    composed.Data.AgentSubject,
		Body:       composed.Data.AgentBody,
		Origin:     shipment.CommentOriginAgent,
		Source:     customerupdateservice.SourceAgentEmail,
	})
}

// renderedEmailPreview is the message a send request would deliver: the
// rendered subject, the plain-text body a reader sees, and its recipients.
func RenderedEmailPreview(
	send *emailports.SendEmailRequest,
	versionID *pulid.ID,
) *agentpreview.MessagePreview {
	message := &agentpreview.MessagePreview{
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

func PlannedEmailPreview(planned *emailports.PlannedEmail) *agentpreview.MessagePreview {
	return &agentpreview.MessagePreview{
		Channel:           agentpreview.MessageChannelEmail,
		To:                planned.To,
		Cc:                planned.CC,
		Bcc:               planned.BCC,
		Attachments:       planned.Attachments,
		Subject:           planned.Subject,
		Body:              planned.Text,
		TemplateVersionID: planned.TemplateVersionID,
	}
}
