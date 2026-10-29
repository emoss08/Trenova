package masterdataquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial/hazardousmaterialdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/holdreasonports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

/*
The reference data an operator names in passing.

"Book it as a flatbed", "which fleet is Ortiz on", "is that commodity hazmat" —
every one of those is a lookup the assistant could not do, so it either guessed
an id or asked the person to go and find one. These are small tables, they
change rarely, and each is a few lines here because the shape is already
generated.

They earn their place by turning a name into an id. Half the write tools take a
serviceTypeId or an equipmentTypeId, and without these the model has no honest
way to get one.
*/

// codedRow is the shape almost every reference table reduces to: something to
// say, something to match, and whether it is still in use.
type codedRow struct {
	ID          string `json:"id"`
	Code        string `json:"code,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

// codedFields are the filters every status/code/description table shares.
func codedFields() []filtercatalog.Field {
	return []filtercatalog.Field{
		{
			Name:   querytool.ListFieldStatus,
			Kind:   filtercatalog.KindEnum,
			Values: querytool.StatusValues,
		},
		{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
		{Name: listFieldDescription, Kind: filtercatalog.KindText},
		{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
	}
}

func NewListEquipmentTypesTool(
	repo equipmenttypeports.EquipmentTypeRepository,
) agenttoolports.AgentQueryTool {
	fields := append(codedFields(), filtercatalog.Field{
		Name:   "class",
		Kind:   filtercatalog.KindEnum,
		Values: []string{"Tractor", "Trailer", "Container", "Other"},
	})

	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolListEquipmentTypes,
		Row:          codedRow{},
		EntityPlural: "equipment types",
		Summary:      equipmentTypesSummary,
		Resource:     permission.ResourceEquipmentType,
		Config: listquery.Describe(
			(*equipmenttype.EquipmentType)(nil),
			equipmenttypedb.EquipmentTypesSchema(),
		),
		Fields:        fields,
		PagesByCursor: true,
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, cursor pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(
				ctx,
				&equipmenttypeports.ListEquipmentTypesRequest{Filter: opts, Cursor: cursor},
			)
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.CursorPage(result, func(item *equipmenttype.EquipmentType) any {
				return codedRow{
					ID:          item.ID.String(),
					Code:        item.Code,
					Description: item.Description,
					Status:      string(item.Status),
					Name:        string(item.Class),
				}
			})
		},
	})
}

func NewListFleetCodesTool(repo fleetcodeports.FleetCodeRepository) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolListFleetCodes,
		Row:          codedRow{},
		EntityPlural: "fleet codes",
		Summary:      fleetCodesSummary,
		Resource:     permission.ResourceFleetCode,
		Config: listquery.Describe(
			(*fleetcode.FleetCode)(nil),
			fleetcodedb.FleetCodesSchema(),
		),
		Fields: codedFields(),
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &fleetcodeports.ListFleetCodesRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *fleetcode.FleetCode) any {
				return codedRow{
					ID:          item.ID.String(),
					Code:        item.Code,
					Description: item.Description,
					Status:      string(item.Status),
				}
			}), nil
		},
	})
}

func NewListServiceTypesTool(
	repo servicetypeports.ServiceTypeRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolMasterdataListServiceTypes,
		Row:          codedRow{},
		EntityPlural: "service types",
		Summary:      serviceTypesSummary,
		Resource:     permission.ResourceServiceType,
		Config: listquery.Describe(
			(*servicetype.ServiceType)(nil),
			servicetypedb.ServiceTypesSchema(),
		),
		Fields: codedFields(),
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &servicetypeports.ListServiceTypesRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *servicetype.ServiceType) any {
				return codedRow{
					ID:          item.ID.String(),
					Code:        item.Code,
					Description: item.Description,
					Status:      string(item.Status),
				}
			}), nil
		},
	})
}

func NewListShipmentTypesTool(
	repo shipmenttypeports.ShipmentTypeRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         toolListShipmentTypes,
		Row:          codedRow{},
		EntityPlural: "shipment types",
		Summary:      shipmentTypesSummary,
		Resource:     permission.ResourceShipmentType,
		Config: listquery.Describe(
			(*shipmenttype.ShipmentType)(nil),
			shipmenttypedb.ShipmentTypesSchema(),
		),
		Fields: codedFields(),
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &shipmenttypeports.ListShipmentTypesRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *shipmenttype.ShipmentType) any {
				return codedRow{
					ID:          item.ID.String(),
					Code:        item.Code,
					Description: item.Description,
					Status:      string(item.Status),
				}
			}), nil
		},
	})
}

type commodityRow struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	FreightClass  string `json:"freightClass,omitempty"`
	Hazardous     bool   `json:"hazardous"`
	Stackable     bool   `json:"stackable"`
	Fragile       bool   `json:"fragile"`
	TemperatureOK bool   `json:"temperatureControlled"`
}

func NewListCommoditiesTool(repo commodityports.CommodityRepository) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         toolListCommodities,
		Row:          commodityRow{},
		EntityPlural: pluralCommodities,
		Summary:      commoditiesSummary,
		Resource:     permission.ResourceCommodity,
		Config: listquery.Describe(
			(*commodity.Commodity)(nil),
			commoditydb.CommoditiesSchema(),
		),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ListFieldStatus,
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
			},
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: listFieldDescription, Kind: filtercatalog.KindText},
			{Name: "freightClass", Kind: filtercatalog.KindText},
			{Name: "stackable", Kind: filtercatalog.KindBoolean},
			{Name: "fragile", Kind: filtercatalog.KindBoolean},
			{
				Name: fieldHazardousMaterialID,
				Kind: filtercatalog.KindText,
				Note: "set when the commodity is hazmat; use isnotnull to find all of them",
			},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &commodityports.ListCommodityRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *commodity.Commodity) any {
				return commodityRow{
					ID:            item.ID.String(),
					Name:          item.Name,
					Status:        string(item.Status),
					FreightClass:  string(item.FreightClass),
					Hazardous:     !item.HazardousMaterialID.IsNil(),
					Stackable:     item.Stackable,
					Fragile:       item.Fragile,
					TemperatureOK: item.MinTemperature != nil || item.MaxTemperature != nil,
				}
			}), nil
		},
	})
}

type hazmatRow struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	Class            string `json:"class,omitempty"`
	UNNumber         string `json:"unNumber,omitempty"`
	PackingGroup     string `json:"packingGroup,omitempty"`
	PlacardRequired  bool   `json:"placardRequired"`
	MarinePollutant  bool   `json:"marinePollutant"`
	InhalationHazard bool   `json:"inhalationHazard"`
}

func NewListHazardousMaterialsTool(
	repo hazardousmaterialports.HazardousMaterialRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         toolListHazardousMaterials,
		Row:          hazmatRow{},
		EntityPlural: "hazardous materials",
		Summary:      hazardousMaterialsSummary,
		Resource:     permission.ResourceHazardousMaterial,
		Config: listquery.Describe(
			(*hazardousmaterial.HazardousMaterial)(nil),
			hazardousmaterialdb.HazardousMaterialsSchema(),
		),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ListFieldStatus,
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
			},
			{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: fieldUNNumber, Kind: filtercatalog.KindText, Note: noteUNNumber},
			{Name: "placardRequired", Kind: filtercatalog.KindBoolean},
			{Name: "marinePollutant", Kind: filtercatalog.KindBoolean},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(
				ctx, &hazardousmaterialports.ListHazardousMaterialsRequest{Filter: opts},
			)
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(
				result.Items,
				func(item *hazardousmaterial.HazardousMaterial) any {
					return hazmatRow{
						ID:               item.ID.String(),
						Code:             item.Code,
						Name:             item.Name,
						Class:            string(item.Class),
						UNNumber:         item.UNNumber,
						PackingGroup:     string(item.PackingGroup),
						PlacardRequired:  item.PlacardRequired,
						MarinePollutant:  item.MarinePollutant,
						InhalationHazard: item.InhalationHazard,
					}
				},
			), nil
		},
	})
}

type accessorialRow struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Method      string `json:"method,omitempty"`
	Amount      string `json:"amount,omitempty"`
}

func NewListAccessorialChargesTool(
	repo accessorialchargeports.AccessorialChargeRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         toolListAccessorialCharges,
		Row:          accessorialRow{},
		EntityPlural: "accessorial charges",
		Summary:      accessorialChargesSummary,
		Resource:     permission.ResourceAccessorialCharge,
		Config: listquery.Describe(
			(*accessorialcharge.AccessorialCharge)(nil),
			accessorialchargedb.AccessorialChargesSchema(),
		),
		Fields: []filtercatalog.Field{
			{
				Name:   querytool.ListFieldStatus,
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
			},
			{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
			{Name: listFieldDescription, Kind: filtercatalog.KindText},
			{Name: "amount", Kind: filtercatalog.KindNumber, Sortable: true},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(
				ctx, &accessorialchargeports.ListAccessorialChargeRequest{Filter: opts},
			)
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(
				result.Items,
				func(item *accessorialcharge.AccessorialCharge) any {
					return accessorialRow{
						ID:          item.ID.String(),
						Code:        item.Code,
						Description: item.Description,
						Status:      string(item.Status),
						Method:      string(item.Method),
						Amount:      item.Amount.String(),
					}
				},
			), nil
		},
	})
}

func NewListDocumentTypesTool(
	repo documenttypeports.DocumentTypeRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         toolListDocumentTypes,
		Row:          codedRow{},
		EntityPlural: "document types",
		Summary:      documentTypesSummary,
		Resource:     permission.ResourceDocumentType,
		Config: listquery.Describe(
			(*documenttype.DocumentType)(nil),
			documenttypedb.DocumentTypesSchema(),
		),
		Fields: []filtercatalog.Field{
			{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: listFieldDescription, Kind: filtercatalog.KindText},
			{
				Name: fieldIsSystem,
				Kind: filtercatalog.KindBoolean,
				Note: noteSystemDocumentType,
			},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &documenttypeports.ListDocumentTypesRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *documenttype.DocumentType) any {
				return codedRow{
					ID:          item.ID.String(),
					Code:        item.Code,
					Name:        item.Name,
					Description: item.Description,
				}
			}), nil
		},
	})
}

func NewListLocationCategoriesTool(
	repo locationcategoryports.LocationCategoryRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         querytool.ToolListLocationCategories,
		Row:          codedRow{},
		EntityPlural: "location categories",
		Summary:      locationCategoriesSummary,
		Resource:     permission.ResourceLocationCategory,
		Config: listquery.Describe(
			(*locationcategory.LocationCategory)(nil),
			locationcategorydb.LocationCategoriesSchema(),
		),
		Fields: []filtercatalog.Field{
			{Name: querytool.ListFieldName, Kind: filtercatalog.KindText, Sortable: true},
			{Name: listFieldDescription, Kind: filtercatalog.KindText},
			{Name: "hasSecureParking", Kind: filtercatalog.KindBoolean},
			{Name: "requiresAppointment", Kind: filtercatalog.KindBoolean},
			{Name: "allowsOvernight", Kind: filtercatalog.KindBoolean},
			{Name: "hasRestroom", Kind: filtercatalog.KindBoolean},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(
				ctx, &locationcategoryports.ListLocationCategoriesRequest{Filter: opts},
			)
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(
				result.Items,
				func(item *locationcategory.LocationCategory) any {
					return codedRow{
						ID:          item.ID.String(),
						Name:        item.Name,
						Description: item.Description,
					}
				},
			), nil
		},
	})
}

type holdReasonRow struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Label          string `json:"label"`
	Type           string `json:"type"`
	Description    string `json:"description,omitempty"`
	Severity       string `json:"defaultSeverity,omitempty"`
	BlocksDispatch bool   `json:"blocksDispatch"`
	BlocksDelivery bool   `json:"blocksDelivery"`
	BlocksBilling  bool   `json:"blocksBilling"`
}

func NewListHoldReasonsTool(
	repo holdreasonports.HoldReasonRepository,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         toolListHoldReasons,
		Row:          holdReasonRow{},
		EntityPlural: "hold reasons",
		Summary:      holdReasonsSummary,
		Resource:     permission.ResourceHoldReason,
		Config: listquery.Describe(
			(*holdreason.HoldReason)(nil),
			holdreasondb.HoldReasonsSchema(),
		),
		Fields: []filtercatalog.Field{
			{
				Name: "type",
				Kind: filtercatalog.KindEnum,
				Values: []string{
					"OperationalHold",
					"ComplianceHold",
					"CustomerHold",
					"FinanceHold",
				},
			},
			{Name: querytool.ListFieldCode, Kind: filtercatalog.KindText, Sortable: true},
			{Name: "label", Kind: filtercatalog.KindText, Sortable: true},
			{
				Name: querytool.FieldActive,
				Kind: filtercatalog.KindBoolean,
				Note: noteActiveHoldReason,
			},
			{Name: "defaultBlocksDispatch", Kind: filtercatalog.KindBoolean},
			{Name: "defaultBlocksBilling", Kind: filtercatalog.KindBoolean},
		},
		Fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listtool.Page, error) {
			result, err := repo.List(ctx, &holdreasonports.ListHoldReasonRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			return listtool.OffsetPage(result.Items, func(item *holdreason.HoldReason) any {
				return holdReasonRow{
					ID:             item.ID.String(),
					Code:           item.Code,
					Label:          item.Label,
					Type:           string(item.Type),
					Description:    item.Description,
					Severity:       string(item.DefaultSeverity),
					BlocksDispatch: item.DefaultBlocksDispatch,
					BlocksDelivery: item.DefaultBlocksDelivery,
					BlocksBilling:  item.DefaultBlocksBilling,
				}
			}), nil
		},
	})
}
