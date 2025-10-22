package searchjobs

import (
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
)

type IndexEntityPayload struct {
	temporaltype.BasePayload
	EntityType meilisearchtype.EntityType
	EntityID   pulid.ID
}
