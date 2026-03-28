package documentintelligencejobs

import (
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

type ProcessDocumentIntelligencePayload struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
}

type ProcessDocumentIntelligenceResult struct {
	DocumentID pulid.ID `json:"documentId"`
	Status     string   `json:"status"`
	Kind       string   `json:"kind"`
}

type ReconcileDocumentIntelligencePayload struct {
	temporaltype.BasePayload

	OlderThanSeconds int64 `json:"olderThanSeconds"`
}

type ReconcileDocumentIntelligenceResult struct {
	Queued int `json:"queued"`
}
