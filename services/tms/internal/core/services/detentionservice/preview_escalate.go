package detentionservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/detention"
)

func EscalationSummary(reason string) string {
	return "Escalated to a person: " + strings.TrimSpace(reason)
}

func EscalationEvidence(
	occurrence *detention.DetentionOccurrence,
	reason string,
	now int64,
) *detention.DetentionEvidence {
	return &detention.DetentionEvidence{
		OrganizationID:        occurrence.OrganizationID,
		BusinessUnitID:        occurrence.BusinessUnitID,
		DetentionOccurrenceID: occurrence.ID,
		Kind:                  detention.EvidenceKindStatusChange,
		Source:                detention.EvidenceSourceManual,
		Summary:               EscalationSummary(reason),
		ObservedAt:            now,
		RecordedAt:            now,
	}
}
