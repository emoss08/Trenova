package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	commodityRecordEntity = "commodity"
	paramCommodityID      = "commodityId"
	paramHazmatID         = "hazardousMaterialId"
)

var freightClasses = agenttoolschema.Source("commodity.freightClass", commodity.FreightClasses)

type commodityKeeper interface {
	Get(ctx context.Context, req repositories.GetCommodityByIDRequest) (*commodity.Commodity, error)
	PlanCreate(ctx context.Context, entity *commodity.Commodity) (*commodity.Commodity, error)
	Create(
		ctx context.Context,
		entity *commodity.Commodity,
		actor *serviceports.RequestActor,
	) (*commodity.Commodity, error)
	PlanUpdate(
		ctx context.Context,
		entity *commodity.Commodity,
	) (*serviceports.RecordChange[commodity.Commodity], error)
	Update(
		ctx context.Context,
		entity *commodity.Commodity,
		actor *serviceports.RequestActor,
	) (*commodity.Commodity, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateCommodityStatusRequest,
	) ([]serviceports.RecordChange[commodity.Commodity], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateCommodityStatusRequest,
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
		WeightPerUnit:          decimalText(entity.WeightPerUnit),
		LinearFeetPerUnit:      decimalText(entity.LinearFeetPerUnit),
		LengthPerUnit:          decimalText(entity.LengthPerUnit),
		WidthPerUnit:           decimalText(entity.WidthPerUnit),
		HeightPerUnit:          decimalText(entity.HeightPerUnit),
		MaxQuantityPerShipment: decimalText(entity.MaxQuantityPerShipment),
		LoadingInstructions:    entity.LoadingInstructions,
		Stackable:              entity.Stackable,
		Fragile:                entity.Fragile,
	}
	if entity.HazardousMaterialID.IsNotNil() {
		view.HazardousMaterialID = entity.HazardousMaterialID.String()
	}

	return view
}

var commodityRecord = &masterRecord[commodity.Commodity]{
	kind:     "commodity",
	resource: permission.ResourceCommodity,
	entity:   commodityRecordEntity,
	idParam:  paramCommodityID,
	idsParam: "commodityIds",
	supplier: "from list_commodities",
	label:    func(entity *commodity.Commodity) string { return entity.Name },
	id:       func(entity *commodity.Commodity) pulid.ID { return entity.ID },
	version:  func(entity *commodity.Commodity) int64 { return entity.Version },
	detach: func(entity *commodity.Commodity) {
		entity.HazardousMaterial = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	view: commodityViewOf,
	options: []toolpreview.Option{toolpreview.WithRefs(map[string]permission.Resource{
		paramHazmatID: permission.ResourceHazardousMaterial,
	})},
}

func commodityFields() []masterField[commodity.Commodity] {
	return []masterField[commodity.Commodity]{
		masterText(mdName, "The commodity's name, unique in this organization.",
			mdMaxShortName, true, func(c *commodity.Commodity) *string { return &c.Name }),
		masterText(mdDescription, "What the goods are.", mdMaxNotes, true,
			func(c *commodity.Commodity) *string { return &c.Description }),
		masterOptionalID(paramHazmatID, permission.ResourceHazardousMaterial,
			"The hazardous material it is, from list_hazardous_materials, when the goods "+
				"are hazmat.",
			func(c *commodity.Commodity) *pulid.ID { return &c.HazardousMaterialID }),
		masterEnum("freightClass", "The NMFC freight class.", freightClasses,
			func(c *commodity.Commodity) *commodity.FreightClass { return &c.FreightClass }),
		masterOptionalInt("minTemperature", "The lowest temperature it may be carried at, "+
			"in degrees Fahrenheit.", mdMinTemperature, mdMaxTemperature,
			func(c *commodity.Commodity) **int { return &c.MinTemperature }),
		masterOptionalInt("maxTemperature", "The highest temperature it may be carried at, "+
			"in degrees Fahrenheit.", mdMinTemperature, mdMaxTemperature,
			func(c *commodity.Commodity) **int { return &c.MaxTemperature }),
		masterDecimal("weightPerUnit", "Pounds per unit.",
			func(c *commodity.Commodity) **float64 { return &c.WeightPerUnit }),
		masterDecimal("linearFeetPerUnit", "Linear feet of trailer per unit.",
			func(c *commodity.Commodity) **float64 { return &c.LinearFeetPerUnit }),
		masterDecimal("lengthPerUnit", "Inches long per unit.",
			func(c *commodity.Commodity) **float64 { return &c.LengthPerUnit }),
		masterDecimal("widthPerUnit", "Inches wide per unit.",
			func(c *commodity.Commodity) **float64 { return &c.WidthPerUnit }),
		masterDecimal("heightPerUnit", "Inches high per unit.",
			func(c *commodity.Commodity) **float64 { return &c.HeightPerUnit }),
		masterDecimal("maxQuantityPerShipment", "The most units one shipment may carry.",
			func(c *commodity.Commodity) **float64 { return &c.MaxQuantityPerShipment }),
		masterText("loadingInstructions", "How it is loaded and secured.", mdMaxNotes, false,
			func(c *commodity.Commodity) *string { return &c.LoadingInstructions }),
		masterBool("stackable", "Whether units may be stacked.",
			func(c *commodity.Commodity) *bool { return &c.Stackable }),
		masterBool("fragile", "Whether it is fragile.",
			func(c *commodity.Commodity) *bool { return &c.Fragile }),
	}
}

func newCreateCommodityTool(commodities commodityKeeper) serviceports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[commodity.Commodity]{
		record: commodityRecord,
		name:   "create_commodity",
		description: "Add a commodity shipments can carry, with its freight class, " +
			"temperature range, dimensions and handling. Name the hazardous material it is " +
			"when it is hazmat, and check list_commodities first.",
		rationale: "Adds a commodity inside Trenova that shipments can name; nothing is sent " +
			"and it can be made inactive.",
		fields:      commodityFields(),
		required:    []string{mdName, mdDescription},
		searchTerms: []string{"new commodity", "add commodity", "product", "freight class"},
		policy:      mdInternalAsk,
		fresh: func(tenant pagination.TenantInfo) *commodity.Commodity {
			return &commodity.Commodity{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				Status:         domaintypes.StatusActive,
			}
		},
		plan:   commodities.PlanCreate,
		create: commodities.Create,
	})
}

func newUpdateCommodityTool(commodities commodityKeeper) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[commodity.Commodity]{
		record: commodityRecord,
		name:   "update_commodity",
		description: "Change a commodity: its name, description, freight class, temperature " +
			"range, dimensions, handling or hazardous material. Fields left out keep their " +
			"value. Use update_commodity_status to activate or retire it.",
		rationale: "Changes a commodity inside Trenova; nothing is sent, and a later change " +
			"puts it back.",
		fields:      commodityFields(),
		searchTerms: []string{"edit commodity", "freight class", "temperature range"},
		policy:      mdInternalAsk,
		get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*commodity.Commodity, error) {
			return commodities.Get(ctx, repositories.GetCommodityByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		plan:   commodities.PlanUpdate,
		update: commodities.Update,
	})
}

func newUpdateCommodityStatusTool(commodities commodityKeeper) serviceports.AgentTool {
	return newMasterStatusTool(&masterStatusSpec[commodity.Commodity, domaintypes.Status]{
		record: commodityRecord,
		name:   "update_commodity_status",
		description: "Set one or more commodities Active or Inactive, which decides whether " +
			"shipments can name them. Get the ids from list_commodities.",
		rationale:   "Changes whether commodities are offered for booking inside Trenova; nothing is sent.",
		searchTerms: []string{"deactivate commodity", "retire commodity"},
		statuses:    activeStatuses,
		statusNote:  activeStatusNote,
		policy:      mdInternalStatus,
		plan: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) ([]serviceports.RecordChange[commodity.Commodity], error) {
			return commodities.PlanBulkUpdateStatus(ctx,
				&repositories.BulkUpdateCommodityStatusRequest{
					TenantInfo:   tenant,
					CommodityIDs: ids,
					Status:       status,
				})
		},
		run: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) ([]*commodity.Commodity, error) {
			return commodities.BulkUpdateStatus(ctx,
				&repositories.BulkUpdateCommodityStatusRequest{
					TenantInfo:   tenant,
					CommodityIDs: ids,
					Status:       status,
				})
		},
	})
}

var (
	hazardClasses = agenttoolschema.Source(
		"hazardousMaterial.class",
		hazardousmaterial.HazardousClassValues(),
	)
	packingGroups = agenttoolschema.Source(
		"hazardousMaterial.packingGroup",
		hazardousmaterial.PackingGroupValues(),
	)
)

const (
	hazmatRecordEntity = "hazardous_material"
	hazmatMaxUN        = 4
	hazmatMaxShort     = 20
	hazmatMaxLabels    = 50
)

type hazmatKeeper interface {
	Get(
		ctx context.Context,
		req repositories.GetHazardousMaterialByIDRequest,
	) (*hazardousmaterial.HazardousMaterial, error)
	PlanCreate(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
	) (*hazardousmaterial.HazardousMaterial, error)
	Create(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
		actor *serviceports.RequestActor,
	) (*hazardousmaterial.HazardousMaterial, error)
	PlanUpdate(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
	) (*serviceports.RecordChange[hazardousmaterial.HazardousMaterial], error)
	Update(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
		actor *serviceports.RequestActor,
	) (*hazardousmaterial.HazardousMaterial, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateHazardousMaterialStatusRequest,
	) ([]serviceports.RecordChange[hazardousmaterial.HazardousMaterial], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateHazardousMaterialStatusRequest,
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

var hazmatRecord = &masterRecord[hazardousmaterial.HazardousMaterial]{
	kind:     "hazardous material",
	resource: permission.ResourceHazardousMaterial,
	entity:   hazmatRecordEntity,
	idParam:  paramHazmatID,
	idsParam: "hazardousMaterialIds",
	supplier: "from list_hazardous_materials",
	label:    func(entity *hazardousmaterial.HazardousMaterial) string { return entity.Name },
	id:       func(entity *hazardousmaterial.HazardousMaterial) pulid.ID { return entity.ID },
	version:  func(entity *hazardousmaterial.HazardousMaterial) int64 { return entity.Version },
	detach: func(entity *hazardousmaterial.HazardousMaterial) {
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	view: hazmatViewOf,
}

type hazmatText struct {
	key         string
	description string
	limit       int
	at          func(*hazardousmaterial.HazardousMaterial) *string
}

func hazmatFields() []masterField[hazardousmaterial.HazardousMaterial] {
	fields := []masterField[hazardousmaterial.HazardousMaterial]{
		masterText(mdName, "The material's name, unique in this organization.",
			mdMaxShortName, true,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.Name }),
		masterText(mdDescription, "What the material is.", mdMaxNotes, true,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.Description }),
		masterEnum("class", "The DOT hazard class or division.", hazardClasses,
			func(h *hazardousmaterial.HazardousMaterial) *hazardousmaterial.HazardousClass {
				return &h.Class
			}),
		masterEnum("packingGroup", "The packing group: I is great danger, III minor.",
			packingGroups,
			func(h *hazardousmaterial.HazardousMaterial) *hazardousmaterial.PackingGroup {
				return &h.PackingGroup
			}),
	}
	for _, text := range []hazmatText{
		{"unNumber", "The four-digit UN or NA number, without the prefix.", hazmatMaxUN,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.UNNumber }},
		{"properShippingName", "The proper shipping name from the hazardous materials " +
			"table.", mdMaxNotes,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.ProperShippingName }},
		{"subsidiaryHazardClass", "Any subsidiary hazard class.", hazmatMaxShort,
			func(h *hazardousmaterial.HazardousMaterial) *string {
				return &h.SubsidiaryHazardClass
			}},
		{"ergGuideNumber", "The Emergency Response Guidebook guide number.", mdMaxCode,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.ErgGuideNumber }},
		{"labelCodes", "The label codes packages carry.", hazmatMaxLabels,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.LabelCodes }},
		{"specialProvisions", "The special provision codes, separated by commas.", mdMaxNotes,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.SpecialProvisions }},
		{"handlingInstructions", "How it is handled.", mdMaxNotes,
			func(h *hazardousmaterial.HazardousMaterial) *string {
				return &h.HandlingInstructions
			}},
		{"emergencyContact", "Who to call in an emergency.", mdMaxName,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.EmergencyContact }},
		{"emergencyContactPhoneNumber", "The emergency response phone number.", mdMaxPhone,
			func(h *hazardousmaterial.HazardousMaterial) *string {
				return &h.EmergencyContactPhoneNumber
			}},
		{"quantityThreshold", "The reportable quantity; required when isReportableQuantity.",
			hazmatMaxShort,
			func(h *hazardousmaterial.HazardousMaterial) *string { return &h.QuantityThreshold }},
	} {
		fields = append(fields, masterText(text.key, text.description, text.limit, false, text.at))
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
		fields = append(fields, masterBool(key, flag.description, flag.at))
	}

	return fields
}

var mdHazmatPolicy = masterPolicy{
	defaultTier: mdInternalAsk.defaultTier,
	maxTier:     mdInternalAsk.defaultTier,
	taintHold:   mdTaintHold,
}

func newCreateHazardousMaterialTool(materials hazmatKeeper) serviceports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[hazardousmaterial.HazardousMaterial]{
		record: hazmatRecord,
		name:   "create_hazardous_material",
		description: "Add a hazardous material commodities can name, with its hazard class, " +
			"UN number and packing group. Give the shipping name, placarding and emergency " +
			"contact as the DOT table does; check list_hazardous_materials first, and a " +
			"person always approves it.",
		rationale: "Adds a hazardous material inside Trenova; nothing is sent. Its class, " +
			"placarding and emergency contact decide what drivers carry and post, so a " +
			"person checks every one against the DOT table before it is saved.",
		fields:      hazmatFields(),
		required:    []string{mdName, mdDescription, "class", "packingGroup"},
		searchTerms: []string{"new hazmat", "add hazardous material", "un number", "placard"},
		policy:      mdHazmatPolicy,
		fresh: func(tenant pagination.TenantInfo) *hazardousmaterial.HazardousMaterial {
			return &hazardousmaterial.HazardousMaterial{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				Status:         domaintypes.StatusActive,
			}
		},
		plan:   materials.PlanCreate,
		create: materials.Create,
	})
}

func newUpdateHazardousMaterialTool(materials hazmatKeeper) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[hazardousmaterial.HazardousMaterial]{
		record: hazmatRecord,
		name:   "update_hazardous_material",
		description: "Change a hazardous material: its class, UN number, packing group, " +
			"shipping name, placarding, handling or emergency contact. Fields left out keep " +
			"their value. Use update_hazardous_material_status to activate or retire it.",
		rationale: "Changes a hazardous material inside Trenova; nothing is sent. What " +
			"drivers placard and who they call follows it, so a person approves every change.",
		fields:      hazmatFields(),
		searchTerms: []string{"edit hazmat", "un number", "packing group", "erg guide"},
		policy:      mdHazmatPolicy,
		get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*hazardousmaterial.HazardousMaterial, error) {
			return materials.Get(ctx, repositories.GetHazardousMaterialByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		plan:   materials.PlanUpdate,
		update: materials.Update,
	})
}

func newUpdateHazardousMaterialStatusTool(materials hazmatKeeper) serviceports.AgentTool {
	return newMasterStatusTool(
		&masterStatusSpec[hazardousmaterial.HazardousMaterial, domaintypes.Status]{
			record: hazmatRecord,
			name:   "update_hazardous_material_status",
			description: "Set one or more hazardous materials Active or Inactive, which " +
				"decides whether commodities can name them. Get the ids from " +
				"list_hazardous_materials.",
			rationale: "Changes whether hazardous materials are offered inside Trenova; " +
				"nothing is sent.",
			searchTerms: []string{"deactivate hazmat", "retire hazardous material"},
			statuses:    activeStatuses,
			statusNote:  activeStatusNote,
			policy:      mdInternalStatus,
			plan: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]serviceports.RecordChange[hazardousmaterial.HazardousMaterial], error) {
				return materials.PlanBulkUpdateStatus(ctx,
					&repositories.BulkUpdateHazardousMaterialStatusRequest{
						TenantInfo:           tenant,
						HazardousMaterialIDs: ids,
						Status:               status,
					})
			},
			run: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]*hazardousmaterial.HazardousMaterial, error) {
				return materials.BulkUpdateStatus(ctx,
					&repositories.BulkUpdateHazardousMaterialStatusRequest{
						TenantInfo:           tenant,
						HazardousMaterialIDs: ids,
						Status:               status,
					})
			},
		},
	)
}
