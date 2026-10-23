package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
)

const (
	listFieldName        = "name"
	listFieldDescription = "description"
)

func masterDataReadProviders() []any {
	return []any{
		provideListEquipmentManufacturersTool,
	}
}

const toolListEquipmentManufacturers = "equipment_list_manufacturers"

// equipmentManufacturersSummary is what a model reads about the manufacturer
// list, whichever path it reads through.
const equipmentManufacturersSummary = "List equipment manufacturers — who built the tractors " +
	"and trailers in the fleet, such as Freightliner or Utility. Use it to turn a make a " +
	"person named into the id equipment_create_tractor and equipment_create_trailer need."

// provideListEquipmentManufacturersTool reads the manufacturer list through
// EquipmentManufacturerTable, the operation the manufacturers page sends.
func provideListEquipmentManufacturersTool(
	reader agentports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agentports.AgentQueryTool, error) {
	return newGraphQLListTool(&graphQLListSpec{
		name:         toolListEquipmentManufacturers,
		entityPlural: "equipment manufacturers",
		summary:      equipmentManufacturersSummary,
		resource:     permission.ResourceEquipmentManufacturer,
	}, reader, permissions)
}

// newListEquipmentManufacturersTool is the repository-backed list the
// table composer's filter catalog still describes; the registered tool reads
// through GraphQL.
func newListEquipmentManufacturersTool(
	repo equipmentmanufacturerports.EquipmentManufacturerRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         toolListEquipmentManufacturers,
		entityPlural: "equipment manufacturers",
		summary:      equipmentManufacturersSummary,
		resource:     permission.ResourceEquipmentManufacturer,
		config: querybuilder.GetFieldConfiguration(
			(*equipmentmanufacturer.EquipmentManufacturer)(nil),
		),
		fields: []listField{
			{Name: paramStatus, Kind: filterEnum, Values: statusValues},
			{Name: listFieldName, Kind: filterText, Sortable: true},
			{Name: listFieldDescription, Kind: filterText},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listPage, error) {
			result, err := repo.List(ctx,
				&equipmentmanufacturerports.ListEquipmentManufacturersRequest{Filter: opts})
			if err != nil {
				return listPage{}, err
			}

			return offsetListPage(
				result.Items,
				func(item *equipmentmanufacturer.EquipmentManufacturer) any {
					return codedRow{
						ID:          item.ID.String(),
						Name:        item.Name,
						Description: item.Description,
						Status:      string(item.Status),
					}
				},
			), nil
		},
	})
}
