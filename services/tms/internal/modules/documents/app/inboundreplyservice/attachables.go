package inboundreplyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	messageResourceType  = "inbound_message"
	customerResourceType = "customer"
	carrierResourceType  = "carrier"
)

type AttachablesRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MessageID  pulid.ID
}

type attachableSource struct {
	source documentports.MailDocumentSource
	kind   triage.AttachableSource
}

func (s *Service) Attachables(
	ctx context.Context,
	req *AttachablesRequest,
) ([]*inboundmessageports.ReplyAttachable, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID, false)
	if err != nil {
		return nil, err
	}
	if s.documents == nil {
		return []*inboundmessageports.ReplyAttachable{}, nil
	}

	sources := []attachableSource{{
		source: documentports.MailDocumentSource{
			ResourceType: messageResourceType,
			ResourceID:   message.ID.String(),
		},
		kind: triage.AttachableFromMessage,
	}}
	add := func(resourceType string, id pulid.ID, kind triage.AttachableSource) {
		if id.IsNil() {
			return
		}
		sources = append(sources, attachableSource{
			source: documentports.MailDocumentSource{
				ResourceType: resourceType,
				ResourceID:   id.String(),
			},
			kind: kind,
		})
	}
	add(
		inboundmessageports.DocumentResourceType,
		message.MatchedShipmentID,
		triage.AttachableFromShipment,
	)
	add(customerResourceType, message.MatchedCustomerID, triage.AttachableFromCustomer)
	add(carrierResourceType, message.MatchedCarrierID, triage.AttachableFromCarrier)

	requested := make([]documentports.MailDocumentSource, 0, len(sources))
	kinds := make(map[documentports.MailDocumentSource]triage.AttachableSource, len(sources))
	for _, source := range sources {
		requested = append(requested, source.source)
		kinds[source.source] = source.kind
	}

	docs, err := s.documents.Mailable(ctx, &documentports.MailableDocumentsRequest{
		TenantInfo: personTenant(req.TenantInfo, req.UserID),
		Sources:    requested,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*inboundmessageports.ReplyAttachable, 0, len(docs))
	for _, doc := range docs {
		kind, ok := kinds[documentports.MailDocumentSource{
			ResourceType: doc.ResourceType,
			ResourceID:   doc.ResourceID,
		}]
		if !ok {
			continue
		}
		out = append(out, attachableOf(doc, kind))
	}

	return out, nil
}

func attachableOf(
	doc *document.Document,
	kind triage.AttachableSource,
) *inboundmessageports.ReplyAttachable {
	attachable := &inboundmessageports.ReplyAttachable{
		ID:          doc.ID,
		FileName:    doc.MailFileName(),
		ContentType: doc.FileType,
		ByteSize:    doc.FileSize,
		CreatedAt:   doc.CreatedAt,
		Source:      kind,
	}
	if doc.DocumentType != nil {
		attachable.DocumentType = doc.DocumentType.Name
	}

	return attachable
}
