package inboundmessageports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TriageDetentionEvidenceRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
}

type TriageDetentionEvidence struct {
	DocumentID pulid.ID
	FileName   string
	Rendered   bool
}

type TriageDetentionEvidenceSource interface {
	DetentionEvidence(
		ctx context.Context,
		req *TriageDetentionEvidenceRequest,
	) (*TriageDetentionEvidence, error)
}
