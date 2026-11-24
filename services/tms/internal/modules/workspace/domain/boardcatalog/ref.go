package boardcatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
)

type Ref struct {
	Kind recordkind.Kind `json:"kind"`
	ID   pulid.ID        `json:"id"`
}
