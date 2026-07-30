package detentionservice

import (
	"bytes"
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/detentionjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// noticePDFReferenceType names the business object a generated notice PDF is
// about, for the generated-documents audit row.
const noticePDFReferenceType = "detention_occurrence"

// NoticePDF is the rendered notice on paper, carried from the render to the
// send and then to storage.
//
// It exists because the bytes are needed in three places and must be identical
// in all of them: the email attachment the customer opens, the shipment document
// the claim file cites, and the generated-documents row that says which template
// version produced them.
type NoticePDF struct {
	Content  []byte
	FileName string

	// Provenance is the render's source, ids, and content hash with the bytes
	// stripped out. RecordGeneratedDocument wants exactly this, and the workflow
	// that files the document carries it through a Temporal payload where
	// megabytes of PDF have no business being.
	Provenance *services.RenderedDocument
}

// renderNoticePDF prints the notice against the context its email was rendered
// from.
//
// Passing the already-built context rather than a reference id is what
// guarantees the attachment and the email quote the same figures. Rendering the
// PDF from a fresh context would re-read the occurrence, which may have moved on
// between the two renders — and a notice whose attachment disagrees with its body
// is worse than no attachment at all.
func (s *Service) renderNoticePDF(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	occurrence *detention.DetentionOccurrence,
	customerID *pulid.ID,
	data any,
) (*NoticePDF, error) {
	rendered, err := s.templates.RenderDocument(ctx, &services.RenderDocumentRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindDetentionNoticePDF,
		CustomerID:  customerID,
		Data:        data,
		ReferenceID: occurrence.ID,
		Title:       noticePDFTitle(occurrence),
	})
	if err != nil {
		return nil, err
	}

	if len(rendered.PDF) == 0 {
		return nil, errortypes.NewBusinessError(
			"The detention notice PDF rendered no content",
		)
	}

	provenance := *rendered
	provenance.PDF = nil
	provenance.HTML = ""

	return &NoticePDF{
		Content:    rendered.PDF,
		FileName:   noticePDFFileName(occurrence),
		Provenance: &provenance,
	}, nil
}

// noticePDFTitle names the document in a reader's PDF viewer and in the file they
// save. A customer filing a dispute should be able to tell two notices apart
// without opening them.
func noticePDFTitle(occurrence *detention.DetentionOccurrence) string {
	ref := occurrence.ShipmentProNumber
	if ref == "" {
		ref = occurrence.ShipmentID.String()
	}

	return "Detention Notice " + ref
}

func noticePDFFileName(occurrence *detention.DetentionOccurrence) string {
	ref := occurrence.ShipmentProNumber
	if ref == "" {
		ref = occurrence.ShipmentID.String()
	}

	return fmt.Sprintf("detention-notice-%s-%s.pdf", ref, occurrence.ID.String())
}

// fileNoticePDF stores the sent notice PDF as a shipment document.
//
// It runs after the email has gone out, and every failure here is logged rather
// than returned: the customer has the notice, the occurrence records that it was
// sent, and the charge stands on that. Losing the filing copy is a bookkeeping
// gap, not a lost notice.
func (s *Service) fileNoticePDF(
	ctx context.Context,
	notice *detention.DetentionNotice,
	occurrence *detention.DetentionOccurrence,
	pdf *NoticePDF,
) {
	log := s.l.With(
		zap.String("operation", "fileNoticePDF"),
		zap.String("noticeId", notice.ID.String()),
	)

	if s.uploadService == nil {
		log.Warn("document uploads are not configured; the notice PDF was not filed")
		return
	}

	filing := &noticePDFFiling{
		TenantInfo: pagination.TenantInfo{
			OrgID:  notice.OrganizationID,
			BuID:   notice.BusinessUnitID,
			UserID: noticeActorUserID(notice),
		},
		Notice:     notice,
		Occurrence: occurrence,
		PDF:        pdf,
	}

	sessionID, err := s.uploadNoticePDF(ctx, filing)
	if err != nil {
		log.Warn("failed to stage the detention notice PDF for storage", zap.Error(err))
		return
	}

	if err = s.startNoticePDFStorage(ctx, filing, sessionID); err != nil {
		log.Warn("failed to start detention notice PDF storage", zap.Error(err))
	}
}

// noticePDFFiling is one notice PDF on its way into document storage.
type noticePDFFiling struct {
	TenantInfo pagination.TenantInfo
	Notice     *detention.DetentionNotice
	Occurrence *detention.DetentionOccurrence
	PDF        *NoticePDF
}

// actor attributes the upload. An automatic notice has no sender, so the system
// owns it either way.
func (f *noticePDFFiling) actor() services.RequestActor {
	return services.RequestActor{
		PrincipalType:  services.PrincipalTypeSystem,
		PrincipalID:    services.SystemPrincipalID,
		UserID:         f.TenantInfo.UserID,
		OrganizationID: f.Notice.OrganizationID,
		BusinessUnitID: f.Notice.BusinessUnitID,
	}
}

func (f *noticePDFFiling) size() int64 {
	return int64(len(f.PDF.Content))
}

// uploadNoticePDF pushes the bytes into an upload session, leaving finalization
// — the document row, the thumbnail, the search projection — to the same
// workflow every other stored document goes through.
func (s *Service) uploadNoticePDF(
	ctx context.Context,
	f *noticePDFFiling,
) (pulid.ID, error) {
	docType, err := s.documentTypeRepo.GetByCode(ctx, repositories.GetDocumentTypeByCodeRequest{
		Code:       documenttype.CodeDetentionNotice,
		TenantInfo: f.TenantInfo,
	})
	if err != nil {
		return pulid.Nil, err
	}

	session, err := s.uploadService.CreateSession(ctx, &services.CreateSessionRequest{
		TenantInfo:     f.TenantInfo,
		Actor:          f.actor(),
		ResourceID:     f.Occurrence.ShipmentID.String(),
		ResourceType:   "shipment",
		FileName:       f.PDF.FileName,
		FileSize:       f.size(),
		ContentType:    "application/pdf",
		Description:    "Detention notice sent to the customer",
		Tags:           []string{"detention", "notice", "generated"},
		DocumentTypeID: docType.ID.String(),
	})
	if err != nil {
		return pulid.Nil, err
	}

	if _, err = s.uploadService.UploadPart(ctx, &services.UploadPartRequest{
		TenantInfo: f.TenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(f.PDF.Content),
		Size:       f.size(),
	}); err != nil {
		return pulid.Nil, err
	}

	return session.ID, nil
}

// startNoticePDFStorage hands the staged upload to the workflow that finalizes
// it, or finalizes it inline where there is no worker to hand it to.
//
// Both paths end in AttachNoticePDFDocument, so the notice and its audit row are
// written by exactly one piece of code regardless of how the document was
// finalized.
func (s *Service) startNoticePDFStorage(
	ctx context.Context,
	f *noticePDFFiling,
	sessionID pulid.ID,
) error {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		session, err := s.uploadService.Complete(ctx, &services.CompletionRequest{
			TenantInfo: f.TenantInfo,
			Actor:      f.actor(),
			SessionID:  sessionID,
		})
		if err != nil {
			return err
		}
		if session.DocumentID == nil || session.DocumentID.IsNil() {
			return errortypes.NewBusinessError(
				"The detention notice PDF upload did not produce a document",
			)
		}

		return s.AttachNoticePDFDocument(ctx, &services.AttachNoticePDFDocumentParams{
			TenantInfo:   f.TenantInfo,
			NoticeID:     f.Notice.ID,
			OccurrenceID: f.Occurrence.ID,
			DocumentID:   *session.DocumentID,
			FileName:     f.PDF.FileName,
			FileSize:     f.size(),
			Provenance:   f.PDF.Provenance,
			UserID:       f.TenantInfo.UserID,
		})
	}

	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			// One workflow per notice: a retried send reuses the id and Temporal
			// rejects the duplicate rather than filing the notice twice.
			ID:        "detention-notice-pdf-" + f.Notice.ID.String(),
			TaskQueue: temporaltype.TaskQueueBilling.String(),
			StaticSummary: fmt.Sprintf(
				"Filing detention notice PDF for notice %s", f.Notice.ID.String(),
			),
		},
		detentionjobs.StoreNoticePDFWorkflowName,
		&detentionjobs.StoreNoticePDFPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: f.Notice.OrganizationID,
				BusinessUnitID: f.Notice.BusinessUnitID,
				UserID:         f.TenantInfo.UserID,
				Timestamp:      s.now(),
			},
			NoticeID:      f.Notice.ID,
			OccurrenceID:  f.Occurrence.ID,
			SessionID:     sessionID,
			FileName:      f.PDF.FileName,
			FileSize:      f.size(),
			Provenance:    f.PDF.Provenance,
			PrincipalType: services.PrincipalTypeSystem,
			PrincipalID:   services.SystemPrincipalID,
		},
	)

	return err
}

// noticeActorUserID names whoever the filing is attributed to. An automatic
// notice has no sender, and the upload session is happy with a nil user — the
// system is the actor either way.
func noticeActorUserID(notice *detention.DetentionNotice) pulid.ID {
	if notice.SentByID != nil && !notice.SentByID.IsNil() {
		return *notice.SentByID
	}

	return pulid.Nil
}

// AttachNoticePDFDocument links a finalized document to its notice and logs it
// against the template version that produced it.
//
// This is the step that makes a disputed detention charge answerable: the notice
// row points at the exact PDF that was attached, and the generated-documents row
// names the template version behind it even after the template has been edited.
func (s *Service) AttachNoticePDFDocument(
	ctx context.Context,
	p *services.AttachNoticePDFDocumentParams,
) error {
	if p == nil || p.NoticeID.IsNil() || p.DocumentID.IsNil() {
		return errortypes.NewValidationError(
			"documentId", errortypes.ErrRequired,
			"A notice id and a document id are required to file a notice PDF")
	}

	if err := s.noticeRepo.AttachPDFDocument(
		ctx,
		&repositories.AttachNoticePDFDocumentRequest{
			TenantInfo: p.TenantInfo,
			NoticeID:   p.NoticeID,
			DocumentID: p.DocumentID,
		},
	); err != nil {
		return err
	}

	if s.templates == nil || p.Provenance == nil {
		return nil
	}

	documentID := p.DocumentID
	if _, err := s.templates.RecordGeneratedDocument(
		ctx,
		&services.RecordGeneratedDocumentRequest{
			TenantInfo:    p.TenantInfo,
			Kind:          documenttemplate.KindDetentionNoticePDF,
			Rendered:      p.Provenance,
			ReferenceType: noticePDFReferenceType,
			ReferenceID:   p.OccurrenceID,
			DocumentID:    &documentID,
			FileName:      p.FileName,
			FileSize:      p.FileSize,
			UserID:        p.UserID,
		},
	); err != nil {
		// The document is filed and the notice points at it. A missing audit row
		// is worth a warning, not an error that would retry the whole filing and
		// duplicate the document.
		s.l.Warn("could not record the generated detention notice PDF",
			zap.String("noticeId", p.NoticeID.String()),
			zap.Error(err))
	}

	return nil
}
