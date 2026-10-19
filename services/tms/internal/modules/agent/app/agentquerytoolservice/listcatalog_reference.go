package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/holdreasonports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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
func codedFields() []listField {
	return []listField{
		{Name: "status", Kind: filterEnum, Values: statusValues},
		{Name: "code", Kind: filterText, Sortable: true},
		{Name: "description", Kind: filterText},
		{Name: "createdAt", Kind: filterDate, Sortable: true},
	}
}

func newListEquipmentTypesTool(
	repo equipmenttypeports.EquipmentTypeRepository,
) agentports.AgentQueryTool {
	fields := append(codedFields(), listField{
		Name:   "class",
		Kind:   filterEnum,
		Values: []string{"Tractor", "Trailer", "Container", "Other"},
	})

	return newListTool(listSpec{
		name:         "equipment_list_types",
		entityPlural: "equipment types",
		summary: "List equipment types — the trailer and tractor classes this " +
			"organization books against, such as dry van, reefer or flatbed. Use it to " +
			"turn an equipment name a person used into the id other tools need.",
		resource: permission.ResourceEquipmentType,
		config:   querybuilder.GetFieldConfiguration((*equipmenttype.EquipmentType)(nil)),
		fields:   fields,
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(
				ctx,
				&equipmenttypeports.ListEquipmentTypesRequest{Filter: opts},
			)
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *equipmenttype.EquipmentType) any {
				return codedRow{
					ID:          item.ID.String(),
					Code:        item.Code,
					Description: item.Description,
					Status:      string(item.Status),
					Name:        string(item.Class),
				}
			}), nil
		},
	})
}

func newListFleetCodesTool(repo fleetcodeports.FleetCodeRepository) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "masterdata_list_fleet_codes",
		entityPlural: "fleet codes",
		summary: "List fleet codes — the groupings drivers and equipment are assigned " +
			"to, each with its own manager and revenue goal.",
		resource: permission.ResourceFleetCode,
		config:   querybuilder.GetFieldConfiguration((*fleetcode.FleetCode)(nil)),
		fields:   codedFields(),
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &fleetcodeports.ListFleetCodesRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *fleetcode.FleetCode) any {
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

func newListServiceTypesTool(
	repo servicetypeports.ServiceTypeRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "masterdata_list_service_types",
		entityPlural: "service types",
		summary: "List service types — the levels of service a shipment can be booked " +
			"at. Use it to turn a service a person named into the id a shipment needs.",
		resource: permission.ResourceServiceType,
		config:   querybuilder.GetFieldConfiguration((*servicetype.ServiceType)(nil)),
		fields:   codedFields(),
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &servicetypeports.ListServiceTypesRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *servicetype.ServiceType) any {
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

func newListShipmentTypesTool(
	repo shipmenttypeports.ShipmentTypeRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "masterdata_list_shipment_types",
		entityPlural: "shipment types",
		summary: "List shipment types — how this organization categorises the freight " +
			"it moves. Use it to turn a type a person named into the id a shipment needs.",
		resource: permission.ResourceShipmentType,
		config:   querybuilder.GetFieldConfiguration((*shipmenttype.ShipmentType)(nil)),
		fields:   codedFields(),
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &shipmenttypeports.ListShipmentTypesRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *shipmenttype.ShipmentType) any {
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

func newListCommoditiesTool(repo commodityports.CommodityRepository) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "masterdata_list_commodities",
		entityPlural: "commodities",
		summary: "List commodities — what this organization hauls, with freight class, " +
			"handling flags and whether the commodity is hazardous. Temperature-controlled " +
			"commodities carry a min or max temperature.",
		resource: permission.ResourceCommodity,
		config:   querybuilder.GetFieldConfiguration((*commodity.Commodity)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "description", Kind: filterText},
			{Name: "freightClass", Kind: filterText},
			{Name: "stackable", Kind: filterBool},
			{Name: "fragile", Kind: filterBool},
			{
				Name: "hazardousMaterialId",
				Kind: filterText,
				Note: "set when the commodity is hazmat; use isnotnull to find all of them",
			},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &commodityports.ListCommodityRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *commodity.Commodity) any {
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

func newListHazardousMaterialsTool(
	repo hazardousmaterialports.HazardousMaterialRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "masterdata_list_hazardous_materials",
		entityPlural: "hazardous materials",
		summary: "List the hazardous materials this organization is set up to haul, with " +
			"hazard class, UN number, packing group and whether a placard is required. " +
			"Use it for questions about hazmat rules on a load, not about which drivers " +
			"hold a hazmat endorsement — that is worker_list.",
		resource: permission.ResourceHazardousMaterial,
		config:   querybuilder.GetFieldConfiguration((*hazardousmaterial.HazardousMaterial)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "unNumber", Kind: filterText, Note: "the UN/NA identification number"},
			{Name: "placardRequired", Kind: filterBool},
			{Name: "marinePollutant", Kind: filterBool},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(
				ctx, &hazardousmaterialports.ListHazardousMaterialsRequest{Filter: opts},
			)
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *hazardousmaterial.HazardousMaterial) any {
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
			}), nil
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

func newListAccessorialChargesTool(
	repo accessorialchargeports.AccessorialChargeRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "masterdata_list_accessorial_charges",
		entityPlural: "accessorial charges",
		summary: "List the accessorial charges this organization bills — detention, " +
			"layover, lumper and the rest — with how each is calculated and at what rate. " +
			"Use it before proposing a charge so the code and amount are the real ones.",
		resource: permission.ResourceAccessorialCharge,
		config:   querybuilder.GetFieldConfiguration((*accessorialcharge.AccessorialCharge)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "description", Kind: filterText},
			{Name: "amount", Kind: filterNumber, Sortable: true},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(
				ctx, &accessorialchargeports.ListAccessorialChargeRequest{Filter: opts},
			)
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *accessorialcharge.AccessorialCharge) any {
				return accessorialRow{
					ID:          item.ID.String(),
					Code:        item.Code,
					Description: item.Description,
					Status:      string(item.Status),
					Method:      string(item.Method),
					Amount:      item.Amount.String(),
				}
			}), nil
		},
	})
}

func newListDocumentTypesTool(
	repo documenttypeports.DocumentTypeRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "document_list_types",
		entityPlural: "document types",
		summary: "List the document types this organization files — rate confirmations, " +
			"bills of lading, proofs of delivery and the rest. Use it to name a document " +
			"correctly when asking for one or attaching one.",
		resource: permission.ResourceDocumentType,
		config:   querybuilder.GetFieldConfiguration((*documenttype.DocumentType)(nil)),
		fields: []listField{
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "description", Kind: filterText},
			{
				Name: "isSystem",
				Kind: filterBool,
				Note: "true for the types Trenova defines and an operator cannot remove",
			},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &documenttypeports.ListDocumentTypesRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *documenttype.DocumentType) any {
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

func newListLocationCategoriesTool(
	repo locationcategoryports.LocationCategoryRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "location_list_categories",
		entityPlural: "location categories",
		summary: "List location categories — how facilities are grouped, and what each " +
			"offers: secure parking, overnight, a restroom, whether an appointment is " +
			"required. Use it for questions about where a driver can stop.",
		resource: permission.ResourceLocationCategory,
		config:   querybuilder.GetFieldConfiguration((*locationcategory.LocationCategory)(nil)),
		fields: []listField{
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "description", Kind: filterText},
			{Name: "hasSecureParking", Kind: filterBool},
			{Name: "requiresAppointment", Kind: filterBool},
			{Name: "allowsOvernight", Kind: filterBool},
			{Name: "hasRestroom", Kind: filterBool},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(
				ctx, &locationcategoryports.ListLocationCategoriesRequest{Filter: opts},
			)
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *locationcategory.LocationCategory) any {
				return codedRow{
					ID:          item.ID.String(),
					Name:        item.Name,
					Description: item.Description,
				}
			}), nil
		},
	})
}

type invoiceRow struct {
	ID               string       `json:"id"`
	Number           string       `json:"number"`
	Status           string       `json:"status"`
	SettlementStatus string       `json:"settlementStatus"`
	DisputeStatus    string       `json:"disputeStatus,omitempty"`
	BillType         string       `json:"billType"`
	BillTo           string       `json:"billTo,omitempty"`
	ProNumber        string       `json:"proNumber,omitempty"`
	TotalAmount      string       `json:"totalAmount,omitempty"`
	Currency         string       `json:"currency,omitempty"`
	InvoiceDate      optionalDate `json:"invoiceDate"`
	// A receivable with no due date is not a receivable that is current; it is
	// one nobody gave terms to. Omitting the key made it look like the former.
	DueDate optionalDate `json:"dueDate"`
}

func newListInvoicesTool(
	repo invoiceports.InvoiceRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "invoice_list",
		entityPlural: "invoices",
		summary: "List invoices narrowed by customer, status, whether they are paid or " +
			"disputed, the amount, or the invoice and due dates. It answers what is unpaid, " +
			"overdue or in dispute.",
		searchTerms: []string{"last invoice", "latest invoice"},
		resource:    permission.ResourceInvoice,
		config:      querybuilder.GetFieldConfiguration((*invoice.Invoice)(nil)),
		fields: []listField{
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: []string{"Draft", "Posted", "Voided"},
				Note:   "where the invoice is in its own lifecycle, not whether it is paid",
			},
			{
				Name:   "settlementStatus",
				Kind:   filterEnum,
				Values: []string{"Unpaid", "PartiallyPaid", "Paid"},
				Note:   "this is the one to use for \"unpaid\" or \"outstanding\"",
			},
			{
				Name:   "disputeStatus",
				Kind:   filterEnum,
				Values: []string{"None", "Disputed"},
			},
			{
				Name:   "billType",
				Kind:   filterEnum,
				Values: []string{"Invoice", "CreditMemo", "DebitMemo"},
				Note:   "CreditMemo and DebitMemo are memos; apply a posted CreditMemo with receivable_apply_credit_memo",
			},
			{
				Name: "id",
				Kind: filterText,
				Note: "in with several ids reads them together",
			},
			{Name: "number", Kind: filterText, Sortable: true},
			{Name: "billToName", Kind: filterText, Note: "who the invoice is billed to"},
			{Name: "shipmentProNumber", Kind: filterText},
			{
				Name:     "dueDate",
				Kind:     filterDate,
				Sortable: true,
				Note:     "overdue is a due date before today with settlementStatus not Paid",
			},
			{Name: "invoiceDate", Kind: filterDate, Sortable: true},
			{Name: "totalAmount", Kind: filterNumber, Sortable: true},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		access: newFieldAccess(permissions),
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			result, err := repo.List(ctx, &invoiceports.ListInvoicesRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			showTotal := gate.show("totalAmount", "totalAmount")

			return listRows(result.Items, func(item *invoice.Invoice) any {
				row := invoiceRow{
					ID:               item.ID.String(),
					Number:           item.Number,
					Status:           string(item.Status),
					SettlementStatus: string(item.SettlementStatus),
					DisputeStatus:    string(item.DisputeStatus),
					BillType:         string(item.BillType),
					BillTo:           item.BillToName,
					ProNumber:        item.ShipmentProNumber,
					Currency:         item.CurrencyCode,
					InvoiceDate:      recordedDate(item.InvoiceDate),
					DueDate:          pointerDate(item.DueDate),
				}
				if showTotal {
					row.TotalAmount = item.TotalAmount.StringFixed(2)
				}

				return row
			}), nil
		},
	})
}

type carrierRow struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	CarrierType      string `json:"carrierType,omitempty"`
	ComplianceStatus string `json:"complianceStatus,omitempty"`
	SafetyRating     string `json:"safetyRating,omitempty"`
	DOTNumber        string `json:"dotNumber,omitempty"`
	MCNumber         string `json:"mcNumber,omitempty"`
}

func newListCarriersTool(repo carrierports.CarrierRepository) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "carrier_list",
		entityPlural: "carriers",
		summary: "List the carriers this organization tenders freight to, narrowed by " +
			"status, compliance, safety rating or authority number. Use it before " +
			"tendering a load to check a carrier is qualified.",
		resource: permission.ResourceCarrier,
		config:   querybuilder.GetFieldConfiguration((*carrier.Carrier)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{
				Name:   "complianceStatus",
				Kind:   filterEnum,
				Values: []string{"Pending", "Qualified", "Disqualified", "Expired"},
				Note:   "only a Qualified carrier should be tendered a load",
			},
			{
				Name:   "safetyRating",
				Kind:   filterEnum,
				Values: []string{"Satisfactory", "Conditional", "Unsatisfactory", "NotRated"},
			},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "dotNumber", Kind: filterText},
			{Name: "mcNumber", Kind: filterText},
			{
				Name:   "carrierType",
				Kind:   filterEnum,
				Values: []string{"Common", "Contract", "Broker", "Exempt"},
			},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &carrierports.ListCarrierRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *carrier.Carrier) any {
				return carrierRow{
					ID:               item.ID.String(),
					Code:             item.Code,
					Name:             item.Name,
					Status:           string(item.Status),
					CarrierType:      string(item.CarrierType),
					ComplianceStatus: string(item.ComplianceStatus),
					SafetyRating:     string(item.SafetyRating),
					DOTNumber:        item.DOTNumber,
					MCNumber:         item.MCNumber,
				}
			}), nil
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

func newListHoldReasonsTool(
	repo holdreasonports.HoldReasonRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "masterdata_list_hold_reasons",
		entityPlural: "hold reasons",
		summary: "List the reasons a shipment can be put on hold, and what each one " +
			"blocks — dispatch, delivery, billing. Call this before shipment_place_hold: " +
			"the reason decides the hold's behaviour, so it has to be one this " +
			"organization actually configured rather than one you describe.",
		resource: permission.ResourceHoldReason,
		config:   querybuilder.GetFieldConfiguration((*holdreason.HoldReason)(nil)),
		fields: []listField{
			{
				Name: "type",
				Kind: filterEnum,
				Values: []string{
					"OperationalHold",
					"ComplianceHold",
					"CustomerHold",
					"FinanceHold",
				},
			},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "label", Kind: filterText, Sortable: true},
			{
				Name: "active",
				Kind: filterBool,
				Note: "only an active reason can be used on a new hold",
			},
			{Name: "defaultBlocksDispatch", Kind: filterBool},
			{Name: "defaultBlocksBilling", Kind: filterBool},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &holdreasonports.ListHoldReasonRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *holdreason.HoldReason) any {
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
