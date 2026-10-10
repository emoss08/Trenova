package documentintelligencecontract

import (
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

type ProcessDocumentIntelligencePayload struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
}
