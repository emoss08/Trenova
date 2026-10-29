package masterdataquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	listFieldDescription = "description"
)

func ReadProviders() []any {
	return []any{
		ProvideListEquipmentManufacturersTool,
	}
}

// equipmentManufacturersSummary is what a model reads about the manufacturer
// list, whichever path it reads through.
const equipmentManufacturersSummary = "List equipment manufacturers — who built the tractors " +
	"and trailers in the fleet, such as Freightliner or Utility. Use it to turn a make a " +
	"person named into the id equipment_create_tractor and equipment_create_trailer need."

// ProvideListEquipmentManufacturersTool reads the manufacturer list through
// EquipmentManufacturerTable, the operation the manufacturers page sends.
func ProvideListEquipmentManufacturersTool(
	reader graphqlreadports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agenttoolports.AgentQueryTool, error) {
	return graphqllisttool.New(&readspec.List{
		Name:         querytool.ToolListEquipmentManufacturers,
		EntityPlural: "equipment manufacturers",
		Summary:      equipmentManufacturersSummary,
		Resource:     permission.ResourceEquipmentManufacturer,
	}, reader, permissions)
}

// NewListEquipmentManufacturersTool is the repository-backed list the
// table composer's filter catalog still describes; the registered tool reads
// through GraphQL.
func NewListEquipmentManufacturersTool(
	repo equipmentmanufacturerports.EquipmentManufacturerRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolListEquipmentManufacturers,
		Row:          codedRow{},
		EntityPlural: "equipment manufacturers",
		Summary:      equipmentManufacturersSummary,
		Resource:     permission.ResourceEquipmentManufacturer,
		Config: listquery.Describe(
			(*equipmentmanufacturer.EquipmentManufacturer)(nil),
			equipmentmanufacturerdb.EquipmentManufacturersSchema(),
		),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ParamStatus,
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
			},
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: listFieldDescription, Kind: filtercatalog.KindText},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx,
				&equipmentmanufacturerports.ListEquipmentManufacturersRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(
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
