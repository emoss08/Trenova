package thumbnailjobs

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type GenerateThumbnailResult struct {
	DocumentID         pulid.ID `json:"documentId"`
	PreviewStoragePath string   `json:"previewStoragePath"`
	Success            bool     `json:"success"`
	Error              string   `json:"error,omitempty"`
}
