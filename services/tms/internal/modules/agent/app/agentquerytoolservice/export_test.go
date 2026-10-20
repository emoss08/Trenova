package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

// The external tests read through the real GraphQL reader, which this
// package cannot import without a cycle; they reach the constructors here.
var (
	ProvideListEquipmentManufacturersTool = provideListEquipmentManufacturersTool
	NewListEquipmentManufacturersTool     = newListEquipmentManufacturersTool
)

var (
	ErrStaleCursor        = errStaleCursor
	ErrOffsetOnCursorList = errOffsetOnCursorList
)

var (
	NewListEquipmentTypesTool     = newListEquipmentTypesTool
	NewListFleetCodesTool         = newListFleetCodesTool
	NewListServiceTypesTool       = newListServiceTypesTool
	NewListShipmentTypesTool      = newListShipmentTypesTool
	NewListCommoditiesTool        = newListCommoditiesTool
	NewListHazardousMaterialsTool = newListHazardousMaterialsTool
	NewListAccessorialChargesTool = newListAccessorialChargesTool
	NewListDocumentTypesTool      = newListDocumentTypesTool
	NewListLocationCategoriesTool = newListLocationCategoriesTool
	NewListHoldReasonsTool        = newListHoldReasonsTool
)

// ReferenceReadNames are the reference lists read through GraphQL.
func ReferenceReadNames() []string {
	specs := referenceReadSpecs()
	names := make([]string, 0, len(specs))
	for idx := range specs {
		names = append(names, specs[idx].name)
	}

	return names
}

// ReferenceReadResource is the resource a reference list reads.
func ReferenceReadResource(name string) permission.Resource {
	specs := referenceReadSpecs()
	for idx := range specs {
		if specs[idx].name == name {
			return specs[idx].resource
		}
	}

	return ""
}

// ReferenceReadTool is the registered reference list of that name.
func ReferenceReadTool(
	name string,
	reader agentports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agentports.AgentQueryTool, error) {
	specs := referenceReadSpecs()
	for idx := range specs {
		if specs[idx].name == name {
			return newGraphQLListTool(&specs[idx], reader, permissions)
		}
	}

	return nil, errGraphQLReadUnbound
}
