package documentintelligencejobs

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	signatureMetricOperation = "signature"
	signedDateNoonOffset     = int64(12 * 60 * 60)
	signedDateFutureSlack    = int64(24 * 60 * 60)
	earliestSignedYear       = 2000
)

type signatureFinding struct {
	Status   document.SignatureStatus
	SignedAt *int64
}

func readSignature(
	reading *documentaiextractionports.AISignatureReading,
	now int64,
) signatureFinding {
	unknown := signatureFinding{Status: document.SignatureStatusUnknown}
	if reading == nil || clampConfidence(reading.Confidence) < reviewReadyConfidenceThreshold {
		return unknown
	}

	switch strings.ToLower(strings.TrimSpace(reading.Status)) {
	case "signed":
		return signatureFinding{
			Status:   document.SignatureStatusSigned,
			SignedAt: readSignedDate(reading.SignedDate, now),
		}
	case "unsigned":
		return signatureFinding{Status: document.SignatureStatusUnsigned}
	default:
		return unknown
	}
}

func readSignedDate(value string, now int64) *int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	day, err := timeutils.ParseCalendarDate(value, time.UTC)
	if err != nil {
		return nil
	}
	if time.Unix(day, 0).UTC().Year() < earliestSignedYear {
		return nil
	}

	signedAt := day + signedDateNoonOffset
	if signedAt > now+signedDateFutureSlack {
		return nil
	}

	return &signedAt
}

func (a *Activities) recordDetectedSignature(
	ctx context.Context,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
	reading *documentaiextractionports.AISignatureReading,
) {
	if reading == nil || doc == nil || doc.DocumentTypeID == nil {
		return
	}
	if !doc.AcceptsDetectedSignature() {
		a.metrics.Document.RecordAIOutcome(signatureMetricOperation, true, "person_set")
		return
	}

	docType, err := a.documentTypeRepo.GetByID(ctx, documenttypeports.GetDocumentTypeByIDRequest{
		ID:         *doc.DocumentTypeID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		a.metrics.Document.RecordAIOutcome(signatureMetricOperation, false, "type_lookup_failed")
		a.logger.Warn("could not read the document type to record its signature",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
		return
	}
	if !docType.RequiresSignature {
		return
	}

	now := timeutils.NowUnix()
	finding := readSignature(reading, now)
	changed, err := a.documentRepo.RecordDetectedSignature(
		ctx,
		&documentports.RecordDetectedSignatureRequest{
			ID:         doc.ID,
			TenantInfo: tenantInfo,
			Status:     finding.Status,
			SignedAt:   finding.SignedAt,
			DetectedAt: now,
		},
	)
	if err != nil {
		a.metrics.Document.RecordAIOutcome(signatureMetricOperation, false, "write_failed")
		a.logger.Warn("could not record the detected document signature",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
		return
	}

	a.metrics.Document.RecordAIOutcome(
		signatureMetricOperation,
		true,
		strings.ToLower(finding.Status.String()),
	)
	if !changed {
		return
	}

	doc.SignatureStatus = finding.Status
	doc.SignedAt = finding.SignedAt
	doc.SignatureSource = document.SignatureSourceExtraction
	a.publishSignatureInvalidation(ctx, doc)
}

func (a *Activities) publishSignatureInvalidation(ctx context.Context, doc *document.Document) {
	if a.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, a.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Resource:       permission.ResourceDocument.String(),
		Action:         string(permission.OpUpdate),
		RecordID:       doc.ID,
		Entity: map[string]string{
			"resourceId":   doc.ResourceID,
			"resourceType": doc.ResourceType,
		},
	}); err != nil {
		a.logger.Warn("failed to publish document signature invalidation",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
	}
}
