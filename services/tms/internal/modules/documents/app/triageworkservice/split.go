package triageworkservice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triageworkjobs/triageworkcontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	pdfContentType     = "application/pdf"
	maxSplitRanges     = 50
	maxSplitFileBytes  = 50 * 1024 * 1024
	maxFileNameRunes   = 200
	maxFiledAsRunes    = 100
	splitDescription   = "Split from an email attachment"
	splitFailedMessage = "Some pages could not be filed. Undo the split and try again."
)

type splitRequest struct {
	tenantInfo pagination.TenantInfo
	userID     pulid.ID
	message    *inboundmessage.InboundMessage
	attachment *inboundmessage.InboundAttachment
	shipmentID pulid.ID
	pages      []PageRange
}

func (s *Service) split(
	ctx context.Context,
	req *splitRequest,
) (*inboundmessage.InboundMessage, error) {
	if err := requireSplittable(req.attachment); err != nil {
		return nil, err
	}

	pdf, err := s.attachmentBytes(ctx, req.tenantInfo, req.attachment)
	if err != nil {
		return nil, err
	}

	pageCount, err := imagingkit.PageCount(
		ctx,
		s.imaging,
		kernelports.ImagingBytes("application/pdf", pdf),
		kernelports.ImagingInteractive,
	)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"pages", errortypes.ErrInvalid, "The file is not a PDF that can be split",
		)
	}
	if req.attachment.PageCount != pageCount {
		req.attachment.PageCount = pageCount
	}

	ranges, err := s.validateRanges(ctx, req.tenantInfo, req.pages, pageCount)
	if err != nil {
		return nil, err
	}

	staged, err := s.stageRanges(ctx, req, pdf, ranges)
	if err != nil {
		return nil, err
	}

	filedAt := s.now()
	applyFiling(req.attachment, req.shipmentID, splitLabel(staged), req.userID, filedAt)
	if _, err = s.messages.UpdateAttachment(ctx, req.attachment); err != nil {
		return nil, err
	}

	payload := &triageworkcontract.SplitFilingPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.tenantInfo.OrgID,
			BusinessUnitID: req.tenantInfo.BuID,
			UserID:         req.userID,
			Timestamp:      filedAt,
		},
		UserID:       req.userID,
		MessageID:    req.message.ID,
		AttachmentID: req.attachment.ID,
		ShipmentID:   req.shipmentID,
		FiledAt:      filedAt,
		Ranges:       staged,
	}

	if s.workflows != nil && s.workflows.Enabled() {
		if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
			ID:            triageworkcontract.SplitFilingWorkflowID(req.attachment.ID, filedAt),
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			StaticSummary: "File " + req.attachment.FileName + " as separate documents",
		}, triageworkcontract.SplitFilingWorkflowName, payload); err != nil {
			s.l.Error("could not start a split filing",
				zap.String("attachmentId", req.attachment.ID.String()), zap.Error(err))

			return nil, s.abandonSplit(ctx, req.attachment, err)
		}

		return s.refreshed(ctx, req.tenantInfo, req.message.ID)
	}

	if err = s.completeInline(ctx, payload); err != nil {
		return nil, err
	}

	return s.refreshed(ctx, req.tenantInfo, req.message.ID)
}

func requireSplittable(attachment *inboundmessage.InboundAttachment) error {
	if !strings.EqualFold(attachment.ContentType, pdfContentType) &&
		!strings.EqualFold(filepath.Ext(attachment.FileName), ".pdf") {
		return errortypes.NewValidationError(
			"pages", errortypes.ErrInvalid, "Only a PDF can be split into page ranges",
		)
	}

	return nil
}

func (s *Service) attachmentBytes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	attachment *inboundmessage.InboundAttachment,
) ([]byte, error) {
	content, err := s.documents.GetDownloadContent(ctx, documentports.GetDocumentByIDRequest{
		ID:         attachment.DocumentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	defer content.Body.Close()

	data, err := io.ReadAll(io.LimitReader(content.Body, maxSplitFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSplitFileBytes {
		return nil, errortypes.NewValidationError(
			"pages", errortypes.ErrInvalid, "The file is too large to split",
		)
	}

	return data, nil
}

type resolvedRange struct {
	PageRange

	docType *documenttype.DocumentType
}

func (s *Service) validateRanges(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pages []PageRange,
	pageCount int,
) ([]resolvedRange, error) {
	if len(pages) > maxSplitRanges {
		return nil, errortypes.NewValidationError(
			"pages", errortypes.ErrInvalid, "Split into at most {0} documents", maxSplitRanges,
		)
	}

	multiErr := errortypes.NewMultiError()
	ordered := slices.Clone(pages)
	for i, page := range ordered {
		field := fmt.Sprintf("pages[%d]", i)
		if page.FromPage < 1 || page.ToPage < page.FromPage || page.ToPage > pageCount {
			multiErr.Add(field+".fromPage", errortypes.ErrInvalid,
				"Pages run from 1 to {0}, first page before last", pageCount)
		}
		if page.DocumentTypeID.IsNil() {
			multiErr.Add(field+".documentTypeId", errortypes.ErrRequired,
				"Choose what to file these pages as")
		}
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	slices.SortFunc(ordered, func(a, b PageRange) int { return a.FromPage - b.FromPage })
	for i := 1; i < len(ordered); i++ {
		if ordered[i].FromPage <= ordered[i-1].ToPage {
			return nil, errortypes.NewValidationError(
				"pages", errortypes.ErrInvalid, "Page ranges cannot overlap",
			)
		}
	}

	types := make(map[pulid.ID]*documenttype.DocumentType, len(ordered))
	resolved := make([]resolvedRange, 0, len(ordered))
	for i, page := range ordered {
		docType, ok := types[page.DocumentTypeID]
		if !ok {
			var err error
			docType, err = s.documentType(
				ctx, tenantInfo, page.DocumentTypeID, fmt.Sprintf("pages[%d].documentTypeId", i),
			)
			if err != nil {
				return nil, err
			}
			types[page.DocumentTypeID] = docType
		}
		resolved = append(resolved, resolvedRange{PageRange: page, docType: docType})
	}

	return resolved, nil
}

func (s *Service) stageRanges(
	ctx context.Context,
	req *splitRequest,
	pdf []byte,
	ranges []resolvedRange,
) ([]triageworkcontract.SplitRange, error) {
	wanted := make([]imagingkit.Range, 0, len(ranges))
	for _, rng := range ranges {
		wanted = append(wanted, imagingkit.Range{First: rng.FromPage - 1, Last: rng.ToPage - 1})
	}
	files, err := imagingkit.ExtractRanges(
		ctx,
		s.imaging,
		kernelports.ImagingBytes("application/pdf", pdf),
		wanted,
		maxSplitFileBytes,
		kernelports.ImagingInteractive,
	)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"pages", errortypes.ErrInvalid, "The file is not a PDF that can be split",
		)
	}

	staged := make([]triageworkcontract.SplitRange, 0, len(ranges))
	for i, rng := range ranges {
		sessionID, stageErr := s.stageRange(ctx, req, rng, files[i])
		if stageErr != nil {
			return nil, stageErr
		}

		staged = append(staged, triageworkcontract.SplitRange{
			FromPage:       rng.FromPage,
			ToPage:         rng.ToPage,
			DocumentTypeID: rng.docType.ID,
			DocumentType:   rng.docType.Name,
			SessionID:      sessionID,
		})
	}

	return staged, nil
}

func (s *Service) stageRange(
	ctx context.Context,
	req *splitRequest,
	rng resolvedRange,
	content []byte,
) (pulid.ID, error) {
	session, err := s.uploads.CreateSession(ctx, &documentuploadports.CreateSessionRequest{
		TenantInfo:        req.tenantInfo,
		Actor:             *permissionports.UserActor(req.tenantInfo),
		ResourceID:        req.shipmentID.String(),
		ResourceType:      inboundmessageports.DocumentResourceType,
		ProcessingProfile: string(document.ProcessingProfileNone),
		FileName:          rangeFileName(req.attachment.FileName, rng.PageRange),
		FileSize:          int64(len(content)),
		ContentType:       pdfContentType,
		Description:       splitDescription,
		Tags:              []string{"inbound", "email", "split"},
		DocumentTypeID:    rng.docType.ID.String(),
	})
	if err != nil {
		return pulid.Nil, err
	}

	if _, err = s.uploads.UploadPart(ctx, &documentuploadports.UploadPartRequest{
		TenantInfo: req.tenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(content),
		Size:       int64(len(content)),
	}); err != nil {
		return pulid.Nil, err
	}

	return session.ID, nil
}

func rangeFileName(original string, rng PageRange) string {
	base := strings.TrimSuffix(original, filepath.Ext(original))
	if strings.TrimSpace(base) == "" {
		base = "Attachment"
	}
	suffix := fmt.Sprintf(" (page %d)", rng.FromPage)
	if rng.ToPage > rng.FromPage {
		suffix = fmt.Sprintf(" (pages %d-%d)", rng.FromPage, rng.ToPage)
	}

	return stringutils.TruncateRunes(base, maxFileNameRunes) + suffix + ".pdf"
}

func splitLabel(staged []triageworkcontract.SplitRange) string {
	names := make([]string, 0, len(staged))
	for _, rng := range staged {
		if !slices.Contains(names, rng.DocumentType) {
			names = append(names, rng.DocumentType)
		}
	}

	return stringutils.TruncateRunes(
		fmt.Sprintf("%s (%s)",
			stringutils.CountNoun(len(staged), "document", "documents"),
			strings.Join(names, ", "),
		),
		maxFiledAsRunes,
	)
}

func (s *Service) abandonSplit(
	ctx context.Context,
	attachment *inboundmessage.InboundAttachment,
	cause error,
) error {
	if err := s.clearFiling(ctx, attachment); err != nil {
		s.l.Error("could not roll back an abandoned split filing",
			zap.String("attachmentId", attachment.ID.String()), zap.Error(err))
	}

	return errortypes.NewBusinessError(splitFailedMessage).WithInternal(cause)
}

func (s *Service) clearFiling(
	ctx context.Context,
	attachment *inboundmessage.InboundAttachment,
) error {
	clearFilingFields(attachment)
	_, err := s.messages.UpdateAttachment(ctx, attachment)

	return err
}

func (s *Service) completeInline(
	ctx context.Context,
	payload *triageworkcontract.SplitFilingPayload,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	record := &triageworkcontract.RecordSplitPayload{
		SplitFilingPayload: *payload,
		Documents:          make([]triageworkcontract.SplitDocument, 0, len(payload.Ranges)),
	}

	for _, rng := range payload.Ranges {
		session, err := s.uploads.Complete(ctx, &documentuploadports.CompletionRequest{
			TenantInfo: tenantInfo,
			Actor:      *permissionports.UserActor(tenantInfo),
			SessionID:  rng.SessionID,
		})
		if err != nil || session.DocumentID == nil || session.DocumentID.IsNil() {
			s.l.Error("could not finalize a split document",
				zap.String("sessionId", rng.SessionID.String()), zap.Error(err))
			record.Failed++

			continue
		}
		record.Documents = append(record.Documents, triageworkcontract.SplitDocument{
			SplitRange: rng,
			DocumentID: *session.DocumentID,
		})
	}

	return s.RecordSplit(ctx, record)
}

func (s *Service) RecordSplit(
	ctx context.Context,
	payload *triageworkcontract.RecordSplitPayload,
) error {
	tenantInfo := pagination.TenantInfo{OrgID: payload.OrganizationID, BuID: payload.BusinessUnitID}
	attachment, err := s.messages.GetAttachmentByID(
		ctx,
		inboundmessageports.GetInboundAttachmentRequest{
			ID:         payload.AttachmentID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return err
	}
	if attachment.FiledAt != payload.FiledAt {
		return nil
	}

	message, err := s.message(ctx, tenantInfo, payload.MessageID)
	if err != nil {
		return err
	}

	if len(payload.Documents) == 0 {
		s.l.Warn("a split filing produced no documents",
			zap.String("attachmentId", attachment.ID.String()))
		if err = s.clearFiling(ctx, attachment); err != nil {
			return err
		}
		s.record(ctx, personActivity(message, triage.ActivityUnfiled, payload.UserID).
			About(attachment.FileName, splitFailedMessage))
		s.changed(ctx, message)

		return nil
	}

	attachment.SplitDocuments = make([]inboundmessage.SplitDocument, 0, len(payload.Documents))
	for _, doc := range payload.Documents {
		attachment.SplitDocuments = append(attachment.SplitDocuments, inboundmessage.SplitDocument{
			FromPage:     doc.FromPage,
			ToPage:       doc.ToPage,
			DocumentID:   doc.DocumentID,
			DocumentType: doc.DocumentType,
		})
	}
	if _, err = s.messages.UpdateAttachment(ctx, attachment); err != nil {
		return err
	}

	detail := stringutils.CountNoun(len(payload.Documents), "document", "documents")
	if payload.Failed > 0 {
		detail += fmt.Sprintf(", %d could not be filed", payload.Failed)
	}
	activity := personActivity(message, triage.ActivitySplit, payload.UserID).
		About(attachment.FileName, detail)
	activity.RecordID = payload.ShipmentID
	s.record(ctx, activity)

	notified := make(map[pulid.ID]struct{}, len(payload.Documents))
	for _, doc := range payload.Documents {
		if _, seen := notified[doc.DocumentTypeID]; seen {
			continue
		}
		notified[doc.DocumentTypeID] = struct{}{}
		s.documentAttached(ctx, tenantInfo, payload.ShipmentID, doc.DocumentTypeID)
	}

	refreshed, err := s.message(ctx, tenantInfo, payload.MessageID)
	if err != nil {
		return err
	}
	s.changed(ctx, refreshed)

	return nil
}
