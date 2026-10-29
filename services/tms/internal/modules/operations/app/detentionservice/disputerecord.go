package detentionservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	disputeRecordResourceType = "shipment"
	disputeRecordContentType  = "application/pdf"
	disputeRecordDescription  = "Detention record prepared for a disputed charge"
)

var disputeRecordTags = []string{"detention", "dispute", "generated"}

type DisputeRecordParams struct {
	TenantInfo pagination.TenantInfo
	Occurrence *detention.DetentionOccurrence
}

type DisputeRecord struct {
	DocumentID pulid.ID
	FileName   string
	Rendered   bool
}

func (s *Service) DisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
) (*DisputeRecord, error) {
	occ := p.Occurrence
	if occ == nil {
		return nil, errortypes.NewValidationError(
			"occurrence", errortypes.ErrRequired, "Detention occurrence is required")
	}

	sent, err := s.sentNoticeDocument(ctx, p)
	if err != nil {
		return nil, err
	}
	if sent.IsNotNil() {
		return &DisputeRecord{DocumentID: sent}, nil
	}

	fileName := disputeRecordFileName(occ)
	existing, err := s.storedDisputeRecord(ctx, p.TenantInfo, occ, fileName)
	if err != nil {
		return nil, err
	}
	if existing.IsNotNil() {
		return &DisputeRecord{DocumentID: existing, FileName: fileName}, nil
	}

	if s.documentStore == nil {
		return nil, nil //nolint:nilnil // nothing can be stored on this deployment
	}

	return s.storeDisputeRecord(ctx, p, fileName)
}

func (s *Service) sentNoticeDocument(
	ctx context.Context,
	p *DisputeRecordParams,
) (pulid.ID, error) {
	notices, err := s.noticeRepo.List(ctx, &detentionports.ListDetentionNoticesRequest{
		OccurrenceID: p.Occurrence.ID,
		TenantInfo:   p.TenantInfo,
	})
	if err != nil {
		return pulid.Nil, err
	}

	var chosen *detentionnotice.DetentionNotice
	for _, notice := range notices {
		if notice.PDFDocumentID == nil || notice.PDFDocumentID.IsNil() ||
			notice.DeliveryStatus.IsFailure() {
			continue
		}
		if chosen == nil || betterRecord(notice, chosen) {
			chosen = notice
		}
	}
	if chosen == nil {
		return pulid.Nil, nil
	}

	return *chosen.PDFDocumentID, nil
}

func betterRecord(candidate, current *detentionnotice.DetentionNotice) bool {
	candidateFinal := candidate.Kind == detentionnotice.NoticeKindFinal
	currentFinal := current.Kind == detentionnotice.NoticeKindFinal
	if candidateFinal != currentFinal {
		return candidateFinal
	}

	return candidate.CreatedAt > current.CreatedAt
}

func (s *Service) storedDisputeRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	occ *detention.DetentionOccurrence,
	fileName string,
) (pulid.ID, error) {
	if s.documents == nil {
		return pulid.Nil, nil
	}

	docs, err := s.documents.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceID:   occ.ShipmentID.String(),
		ResourceType: disputeRecordResourceType,
	})
	if err != nil {
		return pulid.Nil, err
	}

	for _, doc := range docs {
		if doc.OriginalName == fileName && doc.Mailable() {
			return doc.ID, nil
		}
	}

	return pulid.Nil, nil
}

func (s *Service) storeDisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
	fileName string,
) (*DisputeRecord, error) {
	occ := p.Occurrence
	pdf, err := s.renderDisputeRecord(ctx, p)
	if err != nil {
		return nil, err
	}

	docType, err := s.documentTypeRepo.GetByCode(
		ctx,
		documenttypeports.GetDocumentTypeByCodeRequest{
			Code:       documenttype.CodeDetentionNotice,
			TenantInfo: p.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	stored, err := s.documentStore.Store(ctx, &documentports.StoreDocumentRequest{
		TenantInfo: p.TenantInfo,
		Actor: permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeSystem,
			PrincipalID:    permissionports.SystemPrincipalID,
			OrganizationID: p.TenantInfo.OrgID,
			BusinessUnitID: p.TenantInfo.BuID,
		},
		Content:        pdf.Content,
		FileName:       fileName,
		ContentType:    disputeRecordContentType,
		ResourceType:   disputeRecordResourceType,
		ResourceID:     occ.ShipmentID.String(),
		DocumentTypeID: docType.ID,
		Description:    disputeRecordDescription,
		Tags:           disputeRecordTags,
	})
	if err != nil {
		return nil, err
	}

	s.recordDisputeRecord(ctx, p, stored, pdf)

	return &DisputeRecord{DocumentID: stored.ID, FileName: fileName, Rendered: true}, nil
}

func (s *Service) renderDisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
) (*NoticePDF, error) {
	if s.templates == nil || s.contextBuilder == nil {
		return nil, errortypes.NewBusinessError(
			"Detention records cannot be rendered on this deployment",
		)
	}

	occ := p.Occurrence
	location, err := s.tenantLocation(ctx, p.TenantInfo)
	if err != nil {
		return nil, err
	}

	data, err := s.contextBuilder.BuildFrom(ctx, &buildNoticeContextParams{
		Occurrence:    occ,
		Kind:          detentionnotice.NoticeKindFinal,
		FacilityName:  occ.LocationName,
		ShipmentRef:   occ.ShipmentProNumber,
		ArrivalSource: arrivalSourceOf(occ),
		TenantInfo:    p.TenantInfo,
		Location:      location,
	})
	if err != nil {
		return nil, err
	}

	return s.renderNoticePDF(ctx, p.TenantInfo, occ, nil, data)
}

func arrivalSourceOf(occ *detention.DetentionOccurrence) detentionevidence.EvidenceSource {
	for _, evidence := range occ.Evidence {
		if evidence.Kind == detentionevidence.EvidenceKindArrival {
			return evidence.Source
		}
	}

	return ""
}

func (s *Service) recordDisputeRecord(
	ctx context.Context,
	p *DisputeRecordParams,
	stored *document.Document,
	pdf *NoticePDF,
) {
	if pdf.Provenance == nil {
		return
	}

	documentID := stored.ID
	if _, err := s.templates.RecordGeneratedDocument(
		ctx,
		&documenttemplateports.RecordGeneratedDocumentRequest{
			TenantInfo:    p.TenantInfo,
			Kind:          documenttemplate.KindDetentionNoticePDF,
			Rendered:      pdf.Provenance,
			ReferenceType: noticePDFReferenceType,
			ReferenceID:   p.Occurrence.ID,
			DocumentID:    &documentID,
			FileName:      stored.OriginalName,
			FilePath:      stored.StoragePath,
			FileSize:      stored.FileSize,
			Checksum:      stored.ChecksumSHA256,
		},
	); err != nil {
		s.l.Warn("could not record the generated detention record",
			zap.String("occurrenceId", p.Occurrence.ID.String()),
			zap.String("documentId", stored.ID.String()),
			zap.Error(err))
	}
}

func disputeRecordFileName(occ *detention.DetentionOccurrence) string {
	ref := occ.ShipmentProNumber
	if ref == "" {
		ref = occ.ShipmentID.String()
	}

	return fmt.Sprintf("detention-record-%s-%s.pdf", ref, occ.ID.String())
}
