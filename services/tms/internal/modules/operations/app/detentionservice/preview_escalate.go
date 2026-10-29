package detentionservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
)

func EscalationSummary(reason string) string {
	return "Escalated to a person: " + strings.TrimSpace(reason)
}

func EscalationEvidence(
	occurrence *detention.DetentionOccurrence,
	reason string,
	now int64,
) *detentionevidence.DetentionEvidence {
	return &detentionevidence.DetentionEvidence{
		OrganizationID:        occurrence.OrganizationID,
		BusinessUnitID:        occurrence.BusinessUnitID,
		DetentionOccurrenceID: occurrence.ID,
		Kind:                  detentionevidence.EvidenceKindStatusChange,
		Source:                detentionevidence.EvidenceSourceManual,
		Summary:               EscalationSummary(reason),
		ObservedAt:            now,
		RecordedAt:            now,
	}
}
