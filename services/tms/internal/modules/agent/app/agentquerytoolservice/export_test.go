package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
	NewListCustomersTool          = newListCustomersTool
	NewListLocationsTool          = newListLocationsTool
	NewListCarriersTool           = newListCarriersTool
)

// ConvertedListNames are the list tools read through GraphQL by a spec.
func ConvertedListNames() []string {
	specs := graphQLListSpecs()
	names := make([]string, 0, len(specs))
	for idx := range specs {
		names = append(names, specs[idx].name)
	}

	return names
}

// ConvertedListResource is the resource a converted list reads.
func ConvertedListResource(name string) permission.Resource {
	specs := graphQLListSpecs()
	for idx := range specs {
		if specs[idx].name == name {
			return specs[idx].resource
		}
	}

	return ""
}

// ConvertedListTool is the registered converted list of that name.
func ConvertedListTool(
	name string,
	reader agentports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agentports.AgentQueryTool, error) {
	specs := graphQLListSpecs()
	for idx := range specs {
		if specs[idx].name == name {
			return newGraphQLListTool(&specs[idx], reader, permissions)
		}
	}

	return nil, errGraphQLReadUnbound
}

// RunView reads a composed view through a list tool, as compose does.
func RunView(
	ctx context.Context,
	tool agentports.AgentQueryTool,
	params *agentports.QueryToolParams,
	filters []domaintypes.FieldFilter,
	sorting []domaintypes.SortField,
) error {
	_, err := tool.(viewRunner).runView(ctx, params, "", filters, sorting, 5)

	return err
}
