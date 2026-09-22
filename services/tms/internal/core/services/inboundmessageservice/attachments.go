package inboundmessageservice

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// attachmentResourceType is what the document pipeline is told these files
// hang off. It is its own resource rather than "shipment" because at upload
// time nobody knows which shipment it is — that is what reading it decides.
// tenant.DocumentControl maps it onto "shipment" for the draft gate, so a
// tenant that allows shipment drafts allows them on forwarded tenders too.
const attachmentResourceType = "inbound_message"

// AttachmentRef is one file the pipeline still has work to do on.
type AttachmentRef struct {
	MessageID    pulid.ID `json:"messageId"`
	AttachmentID pulid.ID `json:"attachmentId"`
	SessionID    pulid.ID `json:"sessionId"`
	FileName     string   `json:"fileName"`
}

// stageAttachments puts the bytes somewhere before the webhook returns.
//
// This is the one moment the file exists: it arrived base64-encoded inside the
// delivery body and the provider will not send it again. So the upload happens
// on the request path, not in the workflow that reads it — an activity started
// later would have nothing left to upload.
//
// A file the document validator refuses is recorded as refused on its own row.
// The alternative is dropping it, which leaves a message whose attachment count
// disagrees with what the sender attached and no record of why.
func (s *Service) stageAttachments(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	parsed *providerMessage,
) {
	if s.uploads == nil || len(message.Attachments) == 0 {
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: message.OrganizationID,
		BuID:  message.BusinessUnitID,
	}

	for i, row := range message.Attachments {
		if i >= len(parsed.Attachments) {
			break
		}
		content := parsed.Attachments[i].Content
		if len(content) == 0 {
			s.recordAttachmentFailure(ctx, row, "The file arrived empty or could not be decoded.")

			continue
		}

		sessionID, err := s.uploadAttachment(ctx, tenantInfo, message, row, content)
		if err != nil {
			s.l.Warn("could not stage an inbound attachment",
				zap.String("messageId", message.ID.String()),
				zap.String("fileName", row.FileName),
				zap.Error(err))
			s.recordAttachmentFailure(ctx, row, refusalText(err))

			continue
		}

		row.UploadSessionID = sessionID
		if _, err = s.messageRepo.UpdateAttachment(ctx, row); err != nil {
			s.l.Error("staged an inbound attachment but could not record its session",
				zap.String("attachmentId", row.ID.String()), zap.Error(err))
		}
	}
}

func (s *Service) uploadAttachment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	message *inboundmessage.InboundMessage,
	row *inboundmessage.InboundAttachment,
	content []byte,
) (pulid.ID, error) {
	size := int64(len(content))
	actor := services.RequestActor{
		PrincipalType:  services.PrincipalTypeSystem,
		BusinessUnitID: message.BusinessUnitID,
		OrganizationID: message.OrganizationID,
	}

	session, err := s.uploads.CreateSession(ctx, &services.CreateSessionRequest{
		TenantInfo:        tenantInfo,
		Actor:             actor,
		ResourceID:        message.ID.String(),
		ResourceType:      attachmentResourceType,
		ProcessingProfile: string(document.ProcessingProfileInboundAttachment),
		FileName:          row.FileName,
		FileSize:          size,
		ContentType:       row.ContentType,
		Description:       "Attachment from " + message.FromAddress,
		Tags:              []string{"inbound", "email"},
	})
	if err != nil {
		return pulid.Nil, err
	}

	if _, err = s.uploads.UploadPart(ctx, &services.UploadPartRequest{
		TenantInfo: tenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(content),
		Size:       size,
	}); err != nil {
		return pulid.Nil, err
	}

	return session.ID, nil
}

// PendingAttachments are the files on a message that still have a session to
// finalize. One already carrying a document id is left alone, so a workflow
// retry does not upload the same file twice.
func (s *Service) PendingAttachments(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]AttachmentRef, error) {
	rows, err := s.messageRepo.ListAttachments(ctx, messageID, tenantInfo)
	if err != nil {
		return nil, err
	}

	refs := make([]AttachmentRef, 0, len(rows))
	for _, row := range rows {
		if row.UploadSessionID.IsNil() || !row.DocumentID.IsNil() {
			continue
		}
		refs = append(refs, AttachmentRef{
			MessageID:    messageID,
			AttachmentID: row.ID,
			SessionID:    row.UploadSessionID,
			FileName:     row.FileName,
		})
	}

	return refs, nil
}

// RecordAttachmentDocument ties a finalized upload back to the attachment row.
func (s *Service) RecordAttachmentDocument(
	ctx context.Context,
	ref AttachmentRef,
	documentID pulid.ID,
	failureText string,
	tenantInfo pagination.TenantInfo,
) error {
	row, err := s.attachmentByID(ctx, ref, tenantInfo)
	if err != nil {
		return err
	}

	row.DocumentID = documentID
	if failureText != "" {
		row.FailureText = failureText
	}
	_, err = s.messageRepo.UpdateAttachment(ctx, row)

	return err
}

// AttachmentExtractionState is what one poll found.
type AttachmentExtractionState struct {
	Status   document.ContentStatus        `json:"status"`
	Terminal bool                          `json:"terminal"`
	Kind     inboundmessage.AttachmentKind `json:"kind"`
}

// PollAttachmentExtraction reads how far the document pipeline has got.
//
// It polls rather than waiting on the document.extracted event, because that
// event publishes from exactly one place — the draft upsert — so a document
// that fails before reaching it publishes nothing, and a workflow waiting on it
// would hang until its timeout with no record of why.
func (s *Service) PollAttachmentExtraction(
	ctx context.Context,
	ref AttachmentRef,
	tenantInfo pagination.TenantInfo,
) (*AttachmentExtractionState, error) {
	row, err := s.attachmentByID(ctx, ref, tenantInfo)
	if err != nil {
		return nil, err
	}
	if row.DocumentID.IsNil() || s.documents == nil {
		return &AttachmentExtractionState{
			Status:   document.ContentStatusFailed,
			Terminal: true,
			Kind:     inboundmessage.AttachmentUnknown,
		}, nil
	}

	doc, err := s.documents.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         row.DocumentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	state := &AttachmentExtractionState{
		Status: doc.ContentStatus,
		// Indexed is the success terminal, not Extracted: the pipeline writes
		// Extracted only as a way-station and the row it settles on is Indexed.
		Terminal: doc.ContentStatus == document.ContentStatusIndexed ||
			doc.ContentStatus == document.ContentStatusFailed,
		Kind: attachmentKindFor(doc.DetectedKind),
	}
	if !state.Terminal {
		return state, nil
	}

	row.Kind = state.Kind
	if doc.ContentStatus == document.ContentStatusFailed && row.FailureText == "" {
		row.FailureText = "The document could not be read."
	}
	if _, err = s.messageRepo.UpdateAttachment(ctx, row); err != nil {
		s.l.Error("could not record what an inbound attachment turned out to be",
			zap.String("attachmentId", ref.AttachmentID.String()), zap.Error(err))
	}

	return state, nil
}

// GiveUpOnAttachment records that the pipeline ran out of time on a file.
//
// This is not the same as a failed extraction, and it is the case that actually
// happens: EnqueueExtraction returns nil on every one of its gates, so a
// document whose extraction was never started sits at Pending forever with no
// error anywhere. Past the deadline, Pending is a failure, not a wait.
func (s *Service) GiveUpOnAttachment(
	ctx context.Context,
	ref AttachmentRef,
	status document.ContentStatus,
	tenantInfo pagination.TenantInfo,
) error {
	row, err := s.attachmentByID(ctx, ref, tenantInfo)
	if err != nil {
		return err
	}

	row.FailureText = fmt.Sprintf(
		"The document was still %s when the pipeline stopped waiting.",
		strings.ToLower(string(status)),
	)
	if row.Kind == "" {
		row.Kind = inboundmessage.AttachmentUnknown
	}
	_, err = s.messageRepo.UpdateAttachment(ctx, row)

	return err
}

// attachmentByID reads one row through the access path the table is indexed
// for. The reference carries its message because every read of this table is
// scoped by message and tenant, which is what keeps one tenant's attachment id
// from ever resolving against another's rows.
func (s *Service) attachmentByID(
	ctx context.Context,
	ref AttachmentRef,
	tenantInfo pagination.TenantInfo,
) (*inboundmessage.InboundAttachment, error) {
	rows, err := s.messageRepo.ListAttachments(ctx, ref.MessageID, tenantInfo)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.ID == ref.AttachmentID {
			return row, nil
		}
	}

	return nil, fmt.Errorf("attachment %s not found on message %s",
		ref.AttachmentID, ref.MessageID)
}

func (s *Service) recordAttachmentFailure(
	ctx context.Context,
	row *inboundmessage.InboundAttachment,
	text string,
) {
	row.FailureText = text
	row.Kind = inboundmessage.AttachmentUnknown
	if _, err := s.messageRepo.UpdateAttachment(ctx, row); err != nil {
		s.l.Error("could not record why an inbound attachment was refused",
			zap.String("attachmentId", row.ID.String()), zap.Error(err))
	}
}

// attachmentKindFor reads the document classifier's vocabulary into the inbox's.
// An unrecognised kind becomes Other rather than Unknown: the pipeline did read
// the file and did decide something, and saying Unknown would claim otherwise.
func attachmentKindFor(detected string) inboundmessage.AttachmentKind {
	switch strings.ToLower(strings.TrimSpace(detected)) {
	case "":
		return inboundmessage.AttachmentUnknown
	case "rateconfirmation", "rate_confirmation":
		return inboundmessage.AttachmentRateConfirmation
	case "proofofdelivery", "proof_of_delivery", "pod":
		return inboundmessage.AttachmentProofOfDelivery
	case "invoice":
		return inboundmessage.AttachmentInvoice
	case "billoflading", "bill_of_lading", "bol":
		return inboundmessage.AttachmentBillOfLading
	default:
		return inboundmessage.AttachmentOther
	}
}

// refusalText keeps a validator's field-level complaint readable on the row
// without leaking the shape of an internal error to whoever reads the inbox.
func refusalText(err error) string {
	text := err.Error()
	if len(text) > 480 {
		text = text[:480]
	}

	return text
}
