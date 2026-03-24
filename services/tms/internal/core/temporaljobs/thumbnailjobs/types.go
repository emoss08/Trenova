package thumbnailjobs

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type GenerateThumbnailPayload struct {
	DocumentID     pulid.ID `json:"documentId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	StoragePath    string   `json:"storagePath"`
	ContentType    string   `json:"contentType"`
	ResourceType   string   `json:"resourceType"`
}

type GenerateThumbnailResult struct {
	DocumentID         pulid.ID `json:"documentId"`
	PreviewStoragePath string   `json:"previewStoragePath"`
	Success            bool     `json:"success"`
	Error              string   `json:"error,omitempty"`
}
