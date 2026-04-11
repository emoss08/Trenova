package journalsource

import "github.com/emoss08/trenova/shared/pulid"

type Source struct {
	ID                   pulid.ID `json:"id"`
	OrganizationID       pulid.ID `json:"organizationId"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"`
	SourceObjectType     string   `json:"sourceObjectType"`
	SourceObjectID       string   `json:"sourceObjectId"`
	SourceEventType      string   `json:"sourceEventType"`
	SourceDocumentNumber string   `json:"sourceDocumentNumber"`
	Status               string   `json:"status"`
	IdempotencyKey       string   `json:"idempotencyKey"`
	JournalBatchID       pulid.ID `json:"journalBatchId"`
	JournalEntryID       pulid.ID `json:"journalEntryId"`
}
