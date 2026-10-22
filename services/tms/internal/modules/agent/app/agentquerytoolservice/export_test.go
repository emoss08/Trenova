package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// The external tests read through the real GraphQL reader, which this
// package cannot import without a cycle; they reach the constructors here.
var (
	ProvideListEquipmentManufacturersTool = provideListEquipmentManufacturersTool
	NewListEquipmentManufacturersTool     = newListEquipmentManufacturersTool
)

var (
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
	NewListTractorsTool           = newListTractorsTool
	NewListTrailersTool           = newListTrailersTool
	NewListWorkersTool            = newListWorkersTool
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
			return newGraphQLReadTool(&specs[idx], reader, permissions)
		}
	}

	return nil, errGraphQLReadUnbound
}

// MinimalCall is the smallest call a converted list takes: a page of five,
// or, for a list keyed by a record, that record's id.
func MinimalCall(tool agentports.AgentQueryTool) map[string]any {
	keyed, ok := tool.(*graphQLUnpagedListTool)
	if !ok {
		return map[string]any{"limit": 5}
	}
	call := make(map[string]any, len(keyed.required))
	for _, name := range keyed.required {
		call[name] = pulid.MustNew("wrk_").String()
	}

	return call
}

// ExpectedMention is how a converted list's description must name one of its
// parameters, and whether it must name it at all: a parameter whose field is
// hidden at the default data access, or a range's upper end, it need not.
func ExpectedMention(tool agentports.AgentQueryTool, name string) (string, bool) {
	list, ok := tool.(*graphQLListTool)
	if !ok {
		return "", false
	}
	for _, param := range list.params {
		if param.Name != name {
			continue
		}
		mention, described := describedName(param)
		if !described || !list.describable(param) {
			return "", false
		}
		if parent, _, nested := strings.Cut(param.Field, "."); nested {
			mention = stringutils.LowerFirst(strings.TrimPrefix(mention, parent))
		}

		return mention, true
	}

	return "", false
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
