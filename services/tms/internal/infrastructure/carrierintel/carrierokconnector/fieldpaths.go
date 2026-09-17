package carrierokconnector

import (
	"strings"

	"github.com/emoss08/trenova/shared/carrierok"
)

var vendorFieldPaths = map[string]string{
	"dot_number":             "identity.dotNumber",
	"docket_prefix":          "identity.docketPrefix",
	"docket_number":          "identity.docketNumber",
	"docket":                 "identity.docketNumber",
	"legal_name":             "identity.legalName",
	"dba_name":               "identity.dbaName",
	"ein":                    "identity.ein",
	"usdot_status":           "identity.usdotStatus",
	"entity_type_desc":       "identity.entityType",
	"carrier_operation_desc": "identity.carrierOperation",
	"added_date":             "identity.dotAddedAt",
	"dot_age":                "identity.dotAgeDays",

	"authority_common":              "authority.common.status",
	"authority_contract":            "authority.contract.status",
	"authority_broker":              "authority.broker.status",
	"authority_common_pending":      "authority.common.pending",
	"authority_contract_pending":    "authority.contract.pending",
	"authority_broker_pending":      "authority.broker.pending",
	"authority_common_review":       "authority.common.underReview",
	"authority_contract_review":     "authority.contract.underReview",
	"authority_broker_review":       "authority.broker.underReview",
	"authority_common_revocation":   "authority.common.revocationPending",
	"authority_contract_revocation": "authority.contract.revocationPending",
	"authority_broker_revocation":   "authority.broker.revocationPending",
	"authority_age_common":          "authority.common.ageDays",
	"authority_age_contract":        "authority.contract.ageDays",
	"authority_age_broker":          "authority.broker.ageDays",
	"authority_start_common":        "authority.common.grantedAt",
	"authority_start_contract":      "authority.contract.grantedAt",
	"total_revocations":             "authority.totalRevocations",
	"last_revocation_date":          "authority.lastRevocationAt",
	"authority_history":             "authority.history",

	"insurance_bipd_on_file":        "insurance.bipdOnFile",
	"insurance_bipd_required":       "insurance.bipdRequired",
	"insurance_cargo_on_file":       "insurance.cargoOnFile",
	"insurance_cargo_required":      "insurance.cargoRequired",
	"insurance_bond_on_file":        "insurance.bondOnFile",
	"insurance_bond_required":       "insurance.bondRequired",
	"insurance_cancel_count":        "insurance.cancelCount",
	"insurance_last_canceled":       "insurance.lastCanceledAt",
	"insurance_pending_cancel_date": "insurance.pendingCancelAt",
	"insurance_history":             "insurance.filings",

	"safety_rating_desc":      "safety.rating",
	"safety_rating_date":      "safety.ratingDate",
	"iss_value":               "safety.issValue",
	"iss_recommendation":      "safety.issRecommendation",
	"risk_score":              "safety.riskScore",
	"risk_score_probability":  "safety.riskProbability",
	"safety_score":            "safety.safetyScore",
	"out_of_service_flag":     "safety.outOfServiceOrder",
	"out_of_service_date":     "safety.outOfServiceAt",
	"latest_review_type_desc": "safety.latestReviewType",
	"latest_review_date":      "safety.latestReviewAt",

	"inspections_total":                      "inspections.total",
	"inspections_driver":                     "inspections.driver",
	"inspections_vehicle":                    "inspections.vehicle",
	"inspections_hazmat":                     "inspections.hazmat",
	"inspections_driver_out_of_service":      "inspections.driverOos",
	"inspections_vehicle_out_of_service":     "inspections.vehicleOos",
	"inspections_hazmat_out_of_service":      "inspections.hazmatOos",
	"inspections_driver_out_of_service_pct":  "inspections.driverOosRate",
	"inspections_vehicle_out_of_service_pct": "inspections.vehicleOosRate",
	"inspections_hazmat_out_of_service_pct":  "inspections.hazmatOosRate",
	"natl_avg_oos_driver":                    "inspections.nationalDriverOosRate",
	"natl_avg_oos_vehicle":                   "inspections.nationalVehicleOosRate",
	"natl_avg_oos_hazmat":                    "inspections.nationalHazmatOosRate",
	"last_inspection_date":                   "inspections.lastInspectionAt",

	"crashes_total":    "crashes.total",
	"crash_fatalities": "crashes.fatal",
	"crash_injuries":   "crashes.injury",
	"crashes_tow_away": "crashes.tow",
	"last_crash_date":  "crashes.lastCrashAt",

	"total_power_units":    "fleet.powerUnits",
	"total_drivers":        "fleet.drivers",
	"total_drivers_cdl":    "fleet.cdlDrivers",
	"owned_tractors":       "fleet.ownedTractors",
	"term_leased_tractors": "fleet.termLeasedTractors",
	"owned_trailers":       "fleet.ownedTrailers",
	"term_leased_trailers": "fleet.termLeasedTrailers",
	"total_trailers":       "fleet.trailers",
	"total_trucks":         "fleet.trucks",
	"equipment":            "equipment",
	"equipment_history":    "equipment",
	"vehicles":             "equipment",

	"telephone_number":          "contacts.phone",
	"cellphone_number":          "contacts.cellphone",
	"fax_number":                "contacts.fax",
	"email_address":             "contacts.email",
	"company_contact_primary":   "contacts.primaryContact",
	"company_contact_secondary": "contacts.secondaryContact",

	"cargo_carried":                 "operations.cargoCarried",
	"operation_classification_desc": "operations.classification",
	"hazardous_material":            "operations.hazmatCarrier",
	"smartway_flag":                 "operations.smartWay",
	"carbtru_flag":                  "operations.carbCompliant",
	"phmsa_flag":                    "operations.phmsa",
	"mcs150_date":                   "operations.mcs150At",
	"mcs150_mileage":                "operations.mcs150Mileage",
	"boc3_company_name":             "operations.boc3Agent",

	"name_change_count":    "changeHistory.nameChanges",
	"name_last_changed":    "changeHistory.nameLastChangedAt",
	"email_change_count":   "changeHistory.emailChanges",
	"email_last_changed":   "changeHistory.emailLastChangedAt",
	"phone_change_count":   "changeHistory.phoneChanges",
	"phone_last_changed":   "changeHistory.phoneLastChangedAt",
	"address_change_count": "changeHistory.addressChanges",
	"address_last_changed": "changeHistory.addressLastChangedAt",
	"contact_change_count": "changeHistory.contactChanges",
	"contact_last_changed": "changeHistory.contactLastChangedAt",

	"network_graph_count_physical_address":  "network.sharedAddresses",
	"network_graph_count_mailing_address":   "network.sharedAddresses",
	"network_graph_count_telephone_numbers": "network.sharedPhones",
	"network_graph_count_cellphone_numbers": "network.sharedPhones",
	"network_graph_count_email_address":     "network.sharedEmails",
	"network_graph_count_ein":               "network.sharedEins",
	"network_graph_count_equipment":         "network.sharedEquipment",
	"network_graph_physical_address":        "network.links",
	"network_graph_telephone_number":        "network.links",
	"network_graph_email":                   "network.links",
	"network_graph_ein":                     "network.links",
	"network_graph_equipment":               "network.links",

	"total_loads":         "lanes.totalLoads",
	"ftl_percentage":      "lanes.ftlPercent",
	"ltl_percentage":      "lanes.ltlPercent",
	"deadhead_percentage": "lanes.deadheadPercent",
	"first_load_date":     "lanes.firstLoadAt",
	"last_load_date":      "lanes.lastLoadAt",
	"preferred_lanes":     "lanes.preferred",

	"indicator_industry_benchmarks":                   "benchmarks.anyAnomaly",
	"indicator_benchmark_inspection_mileage_ratio":    "benchmarks.inspectionMileageAnomaly",
	"indicator_benchmark_inspected_power_units_ratio": "benchmarks.inspectedUnitsAnomaly",
	"indicator_benchmark_power_unit_mileage_ratio":    "benchmarks.powerUnitMileageAnomaly",
}

var addressFieldPrefixes = [...]struct {
	prefix string
	path   string
}{
	{prefix: "physical_address", path: "identity.physicalAddress"},
	{prefix: "mailing_address", path: "identity.mailingAddress"},
}

var basicFieldPrefixes = [...]struct {
	prefix string
	field  string
}{
	{prefix: "basic_roadside_alert_", field: "roadsideAlert"},
	{prefix: "basic_ac_indicator_", field: "acIndicator"},
	{prefix: "basic_measure_", field: "measure"},
	{prefix: "basic_percentile_", field: "percentile"},
	{prefix: "basic_alert_", field: "alert"},
	{prefix: "intervention_threshold_", field: "threshold"},
}

const misspeltVehicleMaintenance = "vehicle_maintence"

func vendorFieldPath(field string) string {
	key := strings.ToLower(strings.TrimSpace(field))
	if path, ok := vendorFieldPaths[key]; ok {
		return path
	}
	for _, entry := range addressFieldPrefixes {
		if key == entry.prefix || strings.HasPrefix(key, entry.prefix+"_") {
			return entry.path
		}
	}
	for _, entry := range basicFieldPrefixes {
		suffix, found := strings.CutPrefix(key, entry.prefix)
		if !found {
			continue
		}
		if suffix == misspeltVehicleMaintenance {
			suffix = string(carrierok.BasicVehicleMaintenance)
		}
		basic, known := basicCategories[carrierok.BasicCategory(suffix)]
		if !known {
			return ""
		}
		return "basics." + basic.String() + "." + entry.field
	}
	return ""
}
