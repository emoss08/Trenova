package masterdataquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	toolListShipmentTypes      = "masterdata_list_shipment_types"
	toolListCommodities        = "masterdata_list_commodities"
	toolListHazardousMaterials = "masterdata_list_hazardous_materials"
	toolListAccessorialCharges = "masterdata_list_accessorial_charges"
	toolListDocumentTypes      = "document_list_types"
	toolListHoldReasons        = "masterdata_list_hold_reasons"

	pluralCommodities        = "commodities"
	fieldHazardousMaterialID = "hazardousMaterialId"
	fieldUNNumber            = "unNumber"
	fieldIsSystem            = "isSystem"
)

const (
	equipmentTypesSummary = "List equipment types — the trailer and tractor classes this organization books " +
		"against, such as dry van, reefer or flatbed. Use it to turn an equipment name " +
		"a person used into the id other tools need."
	fleetCodesSummary = "List fleet codes — the groupings drivers and equipment are assigned to, each " +
		"with its own manager and revenue goal."
	serviceTypesSummary = "List service types — the levels of service a shipment can be booked at. Use it " +
		"to turn a service a person named into the id a shipment needs."
	shipmentTypesSummary = "List shipment types — how this organization categorises the freight it moves. " +
		"Use it to turn a type a person named into the id a shipment needs."
	commoditiesSummary = "List commodities — what this organization hauls, with freight class, handling " +
		"flags and whether the commodity is hazardous. Temperature-controlled " +
		"commodities carry a min or max temperature."
	hazardousMaterialsSummary = "List the hazardous materials this organization is set up to haul, with hazard " +
		"class, UN number, packing group and whether a placard is required. Use it for " +
		"questions about hazmat rules on a load, not about which drivers hold a hazmat " +
		"endorsement — that is worker_list."
	accessorialChargesSummary = "List the accessorial charges this organization bills — detention, layover, " +
		"lumper and the rest — with how each is calculated and at what rate. Use it " +
		"before proposing a charge so the code and amount are the real ones."
	documentTypesSummary = "List the document types this organization files — rate confirmations, bills of " +
		"lading, proofs of delivery and the rest. Use it to name a document correctly " +
		"when asking for one or attaching one."
	locationCategoriesSummary = "List location categories — how facilities are grouped, and what each offers: " +
		"secure parking, overnight, a restroom, whether an appointment is required. Use " +
		"it for questions about where a driver can stop."
	holdReasonsSummary = "List the reasons a shipment can be put on hold, and what each one blocks — " +
		"dispatch, delivery, billing. Call this before shipment_place_hold: the reason " +
		"decides the hold's behaviour, so it has to be one this organization actually " +
		"configured rather than one you describe."
	noteHazardousCommodity = "A commodity with a hazardous material is hazmat."
	noteUNNumber           = "The UN/NA identification number."
	noteSystemDocumentType = "true for the types Trenova defines and an operator cannot remove."
	noteActiveHoldReason   = "Only an active reason can be used on a new hold."
)

// ReferenceReadSpecs are the reference lists an operator names in passing,
// read through the operation each one's page sends. The repository-backed
// constructors in listcatalog_reference.go remain for the table composer's
// filter catalog.
func ReferenceReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         querytool.ToolListEquipmentTypes,
			EntityPlural: "equipment types",
			Summary:      equipmentTypesSummary,
			Resource:     permission.ResourceEquipmentType,
		},
		{
			Name:         querytool.ToolListFleetCodes,
			EntityPlural: "fleet codes",
			Summary:      fleetCodesSummary,
			Resource:     permission.ResourceFleetCode,
		},
		{
			Name:         querytool.ToolMasterdataListServiceTypes,
			EntityPlural: "service types",
			Summary:      serviceTypesSummary,
			Resource:     permission.ResourceServiceType,
		},
		{
			Name:         toolListShipmentTypes,
			EntityPlural: "shipment types",
			Summary:      shipmentTypesSummary,
			Resource:     permission.ResourceShipmentType,
		},
		{
			Name:         toolListCommodities,
			EntityPlural: pluralCommodities,
			Summary:      commoditiesSummary,
			Resource:     permission.ResourceCommodity,
			Notes:        map[string]string{"hasHazardousMaterial": noteHazardousCommodity},
			References: map[string]readspec.Reference{fieldHazardousMaterialID: {
				Resource: permission.ResourceHazardousMaterial,
				What:     "The hazardous material the commodity is",
				Source:   toolListHazardousMaterials,
			}},
		},
		{
			Name:         toolListHazardousMaterials,
			EntityPlural: "hazardous materials",
			Summary:      hazardousMaterialsSummary,
			Resource:     permission.ResourceHazardousMaterial,
			Notes:        map[string]string{fieldUNNumber: noteUNNumber},
		},
		{
			Name:         toolListAccessorialCharges,
			EntityPlural: "accessorial charges",
			Summary:      accessorialChargesSummary,
			Resource:     permission.ResourceAccessorialCharge,
		},
		{
			Name:         toolListDocumentTypes,
			EntityPlural: "document types",
			Summary:      documentTypesSummary,
			Resource:     permission.ResourceDocumentType,
			Notes:        map[string]string{fieldIsSystem: noteSystemDocumentType},
		},
		{
			Name:         querytool.ToolListLocationCategories,
			EntityPlural: "location categories",
			Summary:      locationCategoriesSummary,
			Resource:     permission.ResourceLocationCategory,
		},
		{
			Name:         toolListHoldReasons,
			EntityPlural: "hold reasons",
			Summary:      holdReasonsSummary,
			Resource:     permission.ResourceHoldReason,
			Notes:        map[string]string{querytool.FieldActive: noteActiveHoldReason},
		},
	}
}
