package shipmentperms

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func Resources() []*permissionspec.ResourceDefinition {
	return slices.Concat(
		operationsResources(),
		commodityResources(),
		locationResources(),
		referenceDataResources(),
		communicationResources(),
	)
}
