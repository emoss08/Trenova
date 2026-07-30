package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// NoticeSweepResult reports what one pass of the outbound notice sweep did.
type NoticeSweepResult struct {
	Due     int `json:"due"`
	Sent    int `json:"sent"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// AttachNoticePDFDocumentParams files a finalized document against the notice it
// was rendered for.
type AttachNoticePDFDocumentParams struct {
	TenantInfo   pagination.TenantInfo
	NoticeID     pulid.ID
	OccurrenceID pulid.ID
	DocumentID   pulid.ID
	FileName     string
	FileSize     int64

	// Provenance is the render behind the bytes, carried rather than re-derived:
	// resolving the template again could name a version an administrator edited
	// after the notice went out.
	Provenance *RenderedDocument
	UserID     pulid.ID
}

// DetentionNoticeService is the slice of the detention service the Temporal
// worker calls.
//
// It is a port for one specific reason: the detention service starts the workflow
// that files a notice PDF, so it imports the jobs package. The jobs package
// therefore cannot import the service back, and this interface is what the
// activities depend on instead.
type DetentionNoticeService interface {
	SweepNoticesDue(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*NoticeSweepResult, error)

	AttachNoticePDFDocument(
		ctx context.Context,
		p *AttachNoticePDFDocumentParams,
	) error
}
