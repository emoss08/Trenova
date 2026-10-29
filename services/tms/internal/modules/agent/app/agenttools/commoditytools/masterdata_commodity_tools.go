package commoditytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const commodityRecordEntity = "commodity"

const paramCommodityID = "commodityId"

const paramHazmatID = "hazardousMaterialId"

var freightClasses = agenttoolschema.Source("commodity.freightClass", commodity.FreightClasses)

type CommodityKeeper interface {
	Get(
		ctx context.Context,
		req commodityports.GetCommodityByIDRequest,
	) (*commodity.Commodity, error)
	PlanCreate(ctx context.Context, entity *commodity.Commodity) (*commodity.Commodity, error)
	Create(
		ctx context.Context,
		entity *commodity.Commodity,
		actor *permissionports.RequestActor,
	) (*commodity.Commodity, error)
	PlanUpdate(
		ctx context.Context,
		entity *commodity.Commodity,
	) (*kernelports.RecordChange[commodity.Commodity], error)
	Update(
		ctx context.Context,
		entity *commodity.Commodity,
		actor *permissionports.RequestActor,
	) (*commodity.Commodity, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *commodityports.BulkUpdateCommodityStatusRequest,
	) ([]kernelports.RecordChange[commodity.Commodity], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *commodityports.BulkUpdateCommodityStatusRequest,
	) ([]*commodity.Commodity, error)
}

type commodityView struct {
	Name                   string `json:"name"`
	Description            string `json:"description"`
	Status                 string `json:"status"`
	HazardousMaterialID    string `json:"hazardousMaterialId,omitempty"`
	FreightClass           string `json:"freightClass,omitempty"`
	MinTemperature         *int   `json:"minTemperature,omitempty"`
	MaxTemperature         *int   `json:"maxTemperature,omitempty"`
	WeightPerUnit          string `json:"weightPerUnit,omitempty"`
	LinearFeetPerUnit      string `json:"linearFeetPerUnit,omitempty"`
	LengthPerUnit          string `json:"lengthPerUnit,omitempty"`
	WidthPerUnit           string `json:"widthPerUnit,omitempty"`
	HeightPerUnit          string `json:"heightPerUnit,omitempty"`
	MaxQuantityPerShipment string `json:"maxQuantityPerShipment,omitempty"`
	LoadingInstructions    string `json:"loadingInstructions,omitempty"`
	Stackable              bool   `json:"stackable"`
	Fragile                bool   `json:"fragile"`
}

func commodityViewOf(entity *commodity.Commodity, _ map[pulid.ID]string) any {
	view := &commodityView{
		Name:                   entity.Name,
		Description:            entity.Description,
		Status:                 string(entity.Status),
		FreightClass:           string(entity.FreightClass),
		MinTemperature:         entity.MinTemperature,
		MaxTemperature:         entity.MaxTemperature,
		WeightPerUnit:          masterdatakit.DecimalText(entity.WeightPerUnit),
		LinearFeetPerUnit:      masterdatakit.DecimalText(entity.LinearFeetPerUnit),
		LengthPerUnit:          masterdatakit.DecimalText(entity.LengthPerUnit),
		WidthPerUnit:           masterdatakit.DecimalText(entity.WidthPerUnit),
		HeightPerUnit:          masterdatakit.DecimalText(entity.HeightPerUnit),
		MaxQuantityPerShipment: masterdatakit.DecimalText(entity.MaxQuantityPerShipment),
		LoadingInstructions:    entity.LoadingInstructions,
		Stackable:              entity.Stackable,
		Fragile:                entity.Fragile,
	}
	if entity.HazardousMaterialID.IsNotNil() {
		view.HazardousMaterialID = entity.HazardousMaterialID.String()
	}

	return view
}

var commodityRecord = &masterdatakit.Record[commodity.Commodity]{
	Kind:     "commodity",
	Resource: permission.ResourceCommodity,
	Entity:   commodityRecordEntity,
	IDParam:  paramCommodityID,
	IDsParam: "commodityIds",
	Lister:   "masterdata_list_commodities",
	Label:    func(entity *commodity.Commodity) string { return entity.Name },
	ID:       func(entity *commodity.Commodity) pulid.ID { return entity.ID },
	Version:  func(entity *commodity.Commodity) int64 { return entity.Version },
	Detach: func(entity *commodity.Commodity) {
		entity.HazardousMaterial = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	View: commodityViewOf,
	Options: []toolpreview.Option{toolpreview.WithRefs(map[string]permission.Resource{
		paramHazmatID: permission.ResourceHazardousMaterial,
	})},
}

func commodityFields() []masterdatakit.Field[commodity.Commodity] {
	return []masterdatakit.Field[commodity.Commodity]{
		masterdatakit.Text(
			masterdatakit.MdName,
			"The commodity's name, unique in this organization.",
			masterdatakit.MdMaxShortName,
			true,
			func(c *commodity.Commodity) *string { return &c.Name },
		),
		masterdatakit.Text(
			masterdatakit.MdDescription,
			"What the goods are.",
			masterdatakit.MdMaxNotes,
			true,
			func(c *commodity.Commodity) *string { return &c.Description },
		),
		masterdatakit.OptionalID(
			paramHazmatID,
			permission.ResourceHazardousMaterial,
			"The hazardous material it is, from masterdata_list_hazardous_materials, when the goods "+
				"are hazmat.",
			func(c *commodity.Commodity) *pulid.ID { return &c.HazardousMaterialID },
		),
		masterdatakit.Enum("freightClass", "The NMFC freight class.", freightClasses,
			func(c *commodity.Commodity) *commodity.FreightClass { return &c.FreightClass }),
		masterdatakit.OptionalInt("minTemperature", "The lowest temperature it may be carried at, "+
			"in degrees Fahrenheit, below maxTemperature. Leave it out for one setpoint such "+
			"as a reefer held at -10 F, which is the maximum.", masterdatakit.MdMinTemperature, masterdatakit.MdMaxTemperature,
			func(c *commodity.Commodity) **int { return &c.MinTemperature }),
		masterdatakit.OptionalInt(
			"maxTemperature",
			"The highest temperature it may be carried at, "+
				"in degrees Fahrenheit.",
			masterdatakit.MdMinTemperature,
			masterdatakit.MdMaxTemperature,
			func(c *commodity.Commodity) **int { return &c.MaxTemperature },
		),
		masterdatakit.Decimal("weightPerUnit", "Pounds per unit.",
			func(c *commodity.Commodity) **float64 { return &c.WeightPerUnit }),
		masterdatakit.Decimal("linearFeetPerUnit", "Linear feet of trailer per unit.",
			func(c *commodity.Commodity) **float64 { return &c.LinearFeetPerUnit }),
		masterdatakit.Decimal("lengthPerUnit", "Inches long per unit.",
			func(c *commodity.Commodity) **float64 { return &c.LengthPerUnit }),
		masterdatakit.Decimal("widthPerUnit", "Inches wide per unit.",
			func(c *commodity.Commodity) **float64 { return &c.WidthPerUnit }),
		masterdatakit.Decimal("heightPerUnit", "Inches high per unit.",
			func(c *commodity.Commodity) **float64 { return &c.HeightPerUnit }),
		masterdatakit.Decimal("maxQuantityPerShipment", "The most units one shipment may carry.",
			func(c *commodity.Commodity) **float64 { return &c.MaxQuantityPerShipment }),
		masterdatakit.Text(
			"loadingInstructions",
			"How it is loaded and secured.",
			masterdatakit.MdMaxNotes,
			false,
			func(c *commodity.Commodity) *string { return &c.LoadingInstructions },
		),
		masterdatakit.Bool("stackable", "Whether units may be stacked.",
			func(c *commodity.Commodity) *bool { return &c.Stackable }),
		masterdatakit.Bool("fragile", "Whether it is fragile.",
			func(c *commodity.Commodity) *bool { return &c.Fragile }),
	}
}

func NewCreateCommodityTool(commodities CommodityKeeper) agenttoolports.AgentTool {
	return masterdatakit.NewCreateTool(&masterdatakit.CreateSpec[commodity.Commodity]{
		Record: commodityRecord,
		Name:   agentpreset.ToolCreateCommodity,
		Description: "Add a commodity shipments can carry, with its freight class, " +
			"temperature range, dimensions and handling. Name the hazardous material it is " +
			"when it is hazmat, and check masterdata_list_commodities first.",
		Rationale: "Adds a commodity inside Trenova that shipments can name; nothing is sent " +
			"and it can be made inactive.",
		Fields:      commodityFields(),
		Required:    []string{masterdatakit.MdName, masterdatakit.MdDescription},
		SearchTerms: []string{"new commodity", "add commodity", "product", "freight class"},
		Policy:      masterdatakit.MdInternalAsk,
		Fresh: func(tenant pagination.TenantInfo) *commodity.Commodity {
			return &commodity.Commodity{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				Status:         domaintypes.StatusActive,
			}
		},
		Plan:   commodities.PlanCreate,
		Create: commodities.Create,
	})
}

func NewUpdateCommodityTool(commodities CommodityKeeper) agenttoolports.AgentTool {
	return masterdatakit.NewUpdateTool(&masterdatakit.UpdateSpec[commodity.Commodity]{
		Record: commodityRecord,
		Name:   "masterdata_update_commodity",
		Description: "Change a commodity: its name, description, freight class, temperature " +
			"range, dimensions, handling or hazardous material. Fields left out keep their " +
			"value. Use masterdata_set_commodity_status to activate or retire it.",
		Rationale: "Changes a commodity inside Trenova; nothing is sent, and a later change " +
			"puts it back.",
		Fields:      commodityFields(),
		SearchTerms: []string{"edit commodity", "freight class", "temperature range"},
		Policy:      masterdatakit.MdInternalAsk,
		Get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*commodity.Commodity, error) {
			return commodities.Get(ctx, commodityports.GetCommodityByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		Plan:   commodities.PlanUpdate,
		Update: commodities.Update,
	})
}

func NewUpdateCommodityStatusTool(commodities CommodityKeeper) agenttoolports.AgentTool {
	return masterdatakit.NewStatusTool(
		&masterdatakit.StatusSpec[commodity.Commodity, domaintypes.Status]{
			Record: commodityRecord,
			Name:   "masterdata_set_commodity_status",
			Description: "Set one or more commodities Active or Inactive, which decides whether " +
				"shipments can name them. Get the ids from masterdata_list_commodities.",
			Rationale:   "Changes whether commodities are offered for booking inside Trenova; nothing is sent.",
			SearchTerms: []string{"deactivate commodity", "retire commodity"},
			Statuses:    masterdatakit.ActiveStatuses,
			StatusNote:  masterdatakit.ActiveStatusNote,
			Policy:      masterdatakit.MdInternalStatus,
			Plan: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]kernelports.RecordChange[commodity.Commodity], error) {
				return commodities.PlanBulkUpdateStatus(ctx,
					&commodityports.BulkUpdateCommodityStatusRequest{
						TenantInfo:   tenant,
						CommodityIDs: ids,
						Status:       status,
					})
			},
			Run: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]*commodity.Commodity, error) {
				return commodities.BulkUpdateStatus(ctx,
					&commodityports.BulkUpdateCommodityStatusRequest{
						TenantInfo:   tenant,
						CommodityIDs: ids,
						Status:       status,
					})
			},
		},
	)
}

var hazardClasses = agenttoolschema.Source(
	"hazardousMaterial.class",
	hazardousmaterial.HazardousClassValues(),
)

var packingGroups = agenttoolschema.Source(
	"hazardousMaterial.packingGroup",
	hazardousmaterial.PackingGroupValues(),
)

const hazmatRecordEntity = "hazardous_material"

const hazmatMaxUN = 4

const hazmatMaxShort = 20

const hazmatMaxLabels = 50

type HazmatKeeper interface {
	Get(
		ctx context.Context,
		req hazardousmaterialports.GetHazardousMaterialByIDRequest,
	) (*hazardousmaterial.HazardousMaterial, error)
	PlanCreate(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
	) (*hazardousmaterial.HazardousMaterial, error)
	Create(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
		actor *permissionports.RequestActor,
	) (*hazardousmaterial.HazardousMaterial, error)
	PlanUpdate(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
	) (*kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error)
	Update(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
		actor *permissionports.RequestActor,
	) (*hazardousmaterial.HazardousMaterial, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest,
	) ([]kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest,
	) ([]*hazardousmaterial.HazardousMaterial, error)
}

type hazmatView struct {
	Code                        string `json:"code,omitempty"`
	Name                        string `json:"name"`
	Description                 string `json:"description"`
	Status                      string `json:"status"`
	Class                       string `json:"class"`
	UNNumber                    string `json:"unNumber,omitempty"`
	PackingGroup                string `json:"packingGroup"`
	ProperShippingName          string `json:"properShippingName,omitempty"`
	SubsidiaryHazardClass       string `json:"subsidiaryHazardClass,omitempty"`
	ErgGuideNumber              string `json:"ergGuideNumber,omitempty"`
	LabelCodes                  string `json:"labelCodes,omitempty"`
	SpecialProvisions           string `json:"specialProvisions,omitempty"`
	HandlingInstructions        string `json:"handlingInstructions,omitempty"`
	EmergencyContact            string `json:"emergencyContact,omitempty"`
	EmergencyContactPhoneNumber string `json:"emergencyContactPhoneNumber,omitempty"`
	QuantityThreshold           string `json:"quantityThreshold,omitempty"`
	PlacardRequired             bool   `json:"placardRequired"`
	IsReportableQuantity        bool   `json:"isReportableQuantity"`
	MarinePollutant             bool   `json:"marinePollutant"`
	InhalationHazard            bool   `json:"inhalationHazard"`
}

func hazmatViewOf(entity *hazardousmaterial.HazardousMaterial, _ map[pulid.ID]string) any {
	return &hazmatView{
		Code:                        entity.Code,
		Name:                        entity.Name,
		Description:                 entity.Description,
		Status:                      string(entity.Status),
		Class:                       string(entity.Class),
		UNNumber:                    entity.UNNumber,
		PackingGroup:                string(entity.PackingGroup),
		ProperShippingName:          entity.ProperShippingName,
		SubsidiaryHazardClass:       entity.SubsidiaryHazardClass,
		ErgGuideNumber:              entity.ErgGuideNumber,
		LabelCodes:                  entity.LabelCodes,
		SpecialProvisions:           entity.SpecialProvisions,
		HandlingInstructions:        entity.HandlingInstructions,
		EmergencyContact:            entity.EmergencyContact,
		EmergencyContactPhoneNumber: entity.EmergencyContactPhoneNumber,
		QuantityThreshold:           entity.QuantityThreshold,
		PlacardRequired:             entity.PlacardRequired,
		IsReportableQuantity:        entity.IsReportableQuantity,
		MarinePollutant:             entity.MarinePollutant,
		InhalationHazard:            entity.InhalationHazard,
	}
}

var hazmatRecord = &masterdatakit.Record[hazardousmaterial.HazardousMaterial]{
	Kind:     "hazardous material",
	Resource: permission.ResourceHazardousMaterial,
	Entity:   hazmatRecordEntity,
	IDParam:  paramHazmatID,
	IDsParam: "hazardousMaterialIds",
	Lister:   "masterdata_list_hazardous_materials",
	Label:    func(entity *hazardousmaterial.HazardousMaterial) string { return entity.Name },
	ID:       func(entity *hazardousmaterial.HazardousMaterial) pulid.ID { return entity.ID },
	Version:  func(entity *hazardousmaterial.HazardousMaterial) int64 { return entity.Version },
	Detach: func(entity *hazardousmaterial.HazardousMaterial) {
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	View: hazmatViewOf,
}

type hazmatText struct {
	key         string
	description string
	limit       int
	at          func(*hazardousmaterial.HazardousMaterial) *string
}

func hazmatFields() []masterdatakit.Field[hazardousmaterial.HazardousMaterial] {
	fields := []masterdatakit.Field[hazardousmaterial.HazardousMaterial]{
		masterdatakit.Text(
			masterdatakit.MdName,
			"The material's name, unique in this organization.",
			masterdatakit.MdMaxShortName,
			true,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.Name },
		),
		masterdatakit.Text(
			masterdatakit.MdDescription,
			"What the material is.",
			masterdatakit.MdMaxNotes,
			true,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.Description },
		),
		masterdatakit.Enum("class", "The DOT hazard class or division.", hazardClasses,
			func(h *hazardousmaterial.HazardousMaterial) *hazardousmaterial.HazardousClass {
				return &h.Class
			}),
		masterdatakit.Enum("packingGroup", "The packing group: I is great danger, III minor.",
			packingGroups,
			func(h *hazardousmaterial.HazardousMaterial) *hazardousmaterial.PackingGroup {
				return &h.PackingGroup
			}),
	}
	for _, text := range []hazmatText{
		{"unNumber", "The four-digit UN or NA number, without the prefix.", hazmatMaxUN,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.UNNumber }},
		{"properShippingName", "The proper shipping name from the hazardous materials " +
			"table.", masterdatakit.MdMaxNotes,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.ProperShippingName }},
		{"subsidiaryHazardClass", "Any subsidiary hazard class.", hazmatMaxShort,
			func(h *hazardousmaterial.HazardousMaterial) *string {
				return &h.SubsidiaryHazardClass
			}},
		{"ergGuideNumber", "The Emergency Response Guidebook guide number.", masterdatakit.MdMaxCode,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.ErgGuideNumber }},
		{"labelCodes", "The label codes packages carry.", hazmatMaxLabels,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.LabelCodes }},
		{"specialProvisions", "The special provision codes, separated by commas.", masterdatakit.MdMaxNotes,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.SpecialProvisions }},
		{"handlingInstructions", "How it is handled.", masterdatakit.MdMaxNotes,
			func(h *hazardousmaterial.HazardousMaterial) *string {
				return &h.HandlingInstructions
			}},
		{"emergencyContact", "Who to call in an emergency.", masterdatakit.MdMaxName,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.EmergencyContact }},
		{"emergencyContactPhoneNumber", "The emergency response phone number.", masterdatakit.MdMaxPhone,
			func(h *hazardousmaterial.HazardousMaterial) *string {
				return &h.EmergencyContactPhoneNumber
			}},
		{"quantityThreshold", "The reportable quantity; required when isReportableQuantity.",
			hazmatMaxShort,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.QuantityThreshold }},
	} {
		fields = append(
			fields,
			masterdatakit.Text(text.key, text.description, text.limit, false, text.at),
		)
	}
	for key, flag := range map[string]struct {
		description string
		at          func(*hazardousmaterial.HazardousMaterial) *bool
	}{
		"placardRequired": {"Whether a vehicle carrying it must be placarded.",
			func(h *hazardousmaterial.HazardousMaterial) *bool { return &h.PlacardRequired }},
		"isReportableQuantity": {"Whether it is a reportable quantity substance.",
			func(h *hazardousmaterial.HazardousMaterial) *bool { return &h.IsReportableQuantity }},
		"marinePollutant": {"Whether it is a marine pollutant.",
			func(h *hazardousmaterial.HazardousMaterial) *bool { return &h.MarinePollutant }},
		"inhalationHazard": {"Whether it is a poison inhalation hazard.",
			func(h *hazardousmaterial.HazardousMaterial) *bool { return &h.InhalationHazard }},
	} {
		fields = append(fields, masterdatakit.Bool(key, flag.description, flag.at))
	}

	return fields
}

var mdHazmatPolicy = masterdatakit.Policy{
	DefaultTier: masterdatakit.MdInternalAsk.DefaultTier,
	MaxTier:     masterdatakit.MdInternalAsk.DefaultTier,
	TaintHold:   masterdatakit.MdTaintHold,
}

func NewCreateHazardousMaterialTool(materials HazmatKeeper) agenttoolports.AgentTool {
	return masterdatakit.NewCreateTool(
		&masterdatakit.CreateSpec[hazardousmaterial.HazardousMaterial]{
			Record: hazmatRecord,
			Name:   "masterdata_create_hazardous_material",
			Description: "Add a hazardous material commodities can name, with its hazard class, " +
				"UN number and packing group. Give the shipping name, placarding and emergency " +
				"contact as the DOT table does; check masterdata_list_hazardous_materials first, and a " +
				"person always approves it.",
			Rationale: "Adds a hazardous material inside Trenova; nothing is sent. Its class, " +
				"placarding and emergency contact decide what drivers carry and post, so a " +
				"person checks every one against the DOT table before it is saved.",
			Fields: hazmatFields(),
			Required: []string{
				masterdatakit.MdName,
				masterdatakit.MdDescription,
				"class",
				"packingGroup",
			},
			SearchTerms: []string{"new hazmat", "add hazardous material", "un number", "placard"},
			Policy:      mdHazmatPolicy,
			Fresh: func(tenant pagination.TenantInfo) *hazardousmaterial.HazardousMaterial {
				return &hazardousmaterial.HazardousMaterial{
					OrganizationID: tenant.OrgID,
					BusinessUnitID: tenant.BuID,
					Status:         domaintypes.StatusActive,
				}
			},
			Plan:   materials.PlanCreate,
			Create: materials.Create,
		},
	)
}

func NewUpdateHazardousMaterialTool(materials HazmatKeeper) agenttoolports.AgentTool {
	return masterdatakit.NewUpdateTool(
		&masterdatakit.UpdateSpec[hazardousmaterial.HazardousMaterial]{
			Record: hazmatRecord,
			Name:   "masterdata_update_hazardous_material",
			Description: "Change a hazardous material: its class, UN number, packing group, " +
				"shipping name, placarding, handling or emergency contact. Fields left out keep " +
				"their value. Use masterdata_set_hazardous_material_status to activate or retire it.",
			Rationale: "Changes a hazardous material inside Trenova; nothing is sent. What " +
				"drivers placard and who they call follows it, so a person approves every change.",
			Fields:      hazmatFields(),
			SearchTerms: []string{"edit hazmat", "un number", "packing group", "erg guide"},
			Policy:      mdHazmatPolicy,
			Get: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				id pulid.ID,
			) (*hazardousmaterial.HazardousMaterial, error) {
				return materials.Get(ctx, hazardousmaterialports.GetHazardousMaterialByIDRequest{
					ID:         id,
					TenantInfo: tenant,
				})
			},
			Plan:   materials.PlanUpdate,
			Update: materials.Update,
		},
	)
}

func NewUpdateHazardousMaterialStatusTool(materials HazmatKeeper) agenttoolports.AgentTool {
	return masterdatakit.NewStatusTool(
		&masterdatakit.StatusSpec[hazardousmaterial.HazardousMaterial, domaintypes.Status]{
			Record: hazmatRecord,
			Name:   "masterdata_set_hazardous_material_status",
			Description: "Set one or more hazardous materials Active or Inactive, which " +
				"decides whether commodities can name them. Get the ids from " +
				"masterdata_list_hazardous_materials.",
			Rationale: "Changes whether hazardous materials are offered inside Trenova; " +
				"nothing is sent.",
			SearchTerms: []string{"deactivate hazmat", "retire hazardous material"},
			Statuses:    masterdatakit.ActiveStatuses,
			StatusNote:  masterdatakit.ActiveStatusNote,
			Policy:      masterdatakit.MdInternalStatus,
			Plan: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]kernelports.RecordChange[hazardousmaterial.HazardousMaterial], error) {
				return materials.PlanBulkUpdateStatus(ctx,
					&hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest{
						TenantInfo:           tenant,
						HazardousMaterialIDs: ids,
						Status:               status,
					})
			},
			Run: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]*hazardousmaterial.HazardousMaterial, error) {
				return materials.BulkUpdateStatus(ctx,
					&hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest{
						TenantInfo:           tenant,
						HazardousMaterialIDs: ids,
						Status:               status,
					})
			},
		},
	)
}
