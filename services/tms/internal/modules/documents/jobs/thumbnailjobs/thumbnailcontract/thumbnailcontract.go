package thumbnailcontract

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
