package editendermapping

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
)

type MappingPreview struct {
	Resolved   []editransfer.MappingResolution `json:"resolved"`
	Unresolved []editransfer.MappingResolution `json:"unresolved"`
	All        []editransfer.MappingResolution `json:"all"`
}
