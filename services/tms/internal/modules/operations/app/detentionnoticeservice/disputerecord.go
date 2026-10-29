package detentionnoticeservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func betterRecord(candidate, current *detentionnotice.DetentionNotice) bool {
	candidateFinal := candidate.Kind == detentionnotice.NoticeKindFinal
	currentFinal := current.Kind == detentionnotice.NoticeKindFinal
	if candidateFinal != currentFinal {
		return candidateFinal
	}

	return candidate.CreatedAt > current.CreatedAt
}

func arrivalSourceOf(occ *detention.DetentionOccurrence) detentionevidence.EvidenceSource {
	for _, evidence := range occ.Evidence {
		if evidence.Kind == detentionevidence.EvidenceKindArrival {
			return evidence.Source
		}
	}

	return ""
}

func disputeRecordFileName(occ *detention.DetentionOccurrence) string {
	ref := occ.ShipmentProNumber
	if ref == "" {
		ref = occ.ShipmentID.String()
	}

	return fmt.Sprintf("detention-record-%s-%s.pdf", ref, occ.ID.String())
}
