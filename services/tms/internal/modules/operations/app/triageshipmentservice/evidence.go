package triageshipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/ports/triageworkports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticeservice"
)

func (s *Service) DetentionEvidence(
	ctx context.Context,
	req *triageworkports.TriageDetentionEvidenceRequest,
) (*triageworkports.TriageDetentionEvidence, error) {
	occurrence, err := s.disputedOccurrence(ctx, req.TenantInfo, req.ShipmentID, true)
	if err != nil {
		return nil, err
	}
	if occurrence == nil || s.records == nil {
		return nil, nil //nolint:nilnil // the shipment has no detention to show
	}

	record, err := s.records.DisputeRecord(ctx, &detentionnoticeservice.DisputeRecordParams{
		TenantInfo: req.TenantInfo,
		Occurrence: occurrence,
	})
	if err != nil {
		return nil, err
	}
	if record == nil || record.DocumentID.IsNil() {
		return nil, nil //nolint:nilnil // no record could be found or made
	}

	return &triageworkports.TriageDetentionEvidence{
		DocumentID: record.DocumentID,
		FileName:   record.FileName,
		Rendered:   record.Rendered,
	}, nil
}
