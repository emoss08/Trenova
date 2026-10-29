package adminperms

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func Resources() []*permissionspec.ResourceDefinition {
	return slices.Concat(
		administrationResources(),
		protectedFieldResources(),
	)
}
