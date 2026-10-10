package documentuploadcontract

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

type FinalizeUploadPayload struct {
	temporaltype.BasePayload

	SessionID     pulid.ID               `json:"sessionId"`
	PrincipalType services.PrincipalType `json:"principalType,omitempty"`
	PrincipalID   pulid.ID               `json:"principalId,omitempty"`
	APIKeyID      pulid.ID               `json:"apiKeyId,omitempty"`
}

type CleanupDocumentStoragePayload struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
	Paths      []string `json:"paths"`
}
