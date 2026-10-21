package agentquerytoolservice

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

const (
	toolListTractors = "equipment_list_tractors"
	toolListTrailers = "equipment_list_trailers"
	toolWorkerList   = "worker_list"

	pluralTractors = "tractors"
	pluralTrailers = "trailers"

	tractorsSummary = "List tractors (power units) narrowed by status, ownership, or a " +
		"registration or lease date. Use it for questions about which units are out " +
		"of service or coming due for plates."
	trailersSummary = "List trailers narrowed by status, ownership, or a registration, " +
		"lease or inspection date, each with its equipment type (dry van, reefer, " +
		"flatbed). Use it for questions about which trailers are out of service or " +
		"overdue for inspection."
	noteEquipmentStatus = "Available means it can be dispatched, OutOfService means it " +
		"cannot, AtMaintenance means it is in the shop, and Sold means it has left the fleet."
	notePlateLapses    = "When the plate lapses."
	noteLastInspection = "The last annual inspection."

	fieldRegistrationExpiryFrom = "registrationExpiryFrom"
	fieldLastInspectionFrom     = "lastInspectionDateFrom"

	fieldEquipmentTypeID         = "equipmentTypeId"
	fieldEquipmentManufacturerID = "equipmentManufacturerId"
	fieldFleetCodeID             = "fleetCodeId"
	fieldLastKnownLocationID     = "lastKnownLocationId"
)

// equipmentReferences are the ids a tractor or trailer filters by, and the
// tool that supplies each.
func equipmentReferences(workers ...string) map[string]idReference {
	refs := map[string]idReference{
		fieldEquipmentTypeID: {
			resource: permission.ResourceEquipmentType,
			what:     "The equipment type",
			source:   toolListEquipmentTypes,
		},
		fieldEquipmentManufacturerID: {
			resource: permission.ResourceEquipmentManufacturer,
			what:     "The manufacturer",
			source:   toolListEquipmentManufacturers,
		},
		fieldFleetCodeID: {
			resource: permission.ResourceFleetCode,
			what:     "The fleet code",
			source:   toolListFleetCodes,
		},
		fieldLastKnownLocationID: {
			resource: permission.ResourceLocation,
			what:     "The location it was last seen at",
			source:   toolLocationList,
		},
	}
	for _, field := range workers {
		refs[field] = workerReference
	}

	return refs
}

// equipmentReadSpecs are the tractor and trailer lists, read through the
// operation each one's page sends. A tractor's drivers are workers, so a
// reader who cannot read workers sees only their ids, as before.
func equipmentReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolListTractors,
			entityPlural: pluralTractors,
			summary:      tractorsSummary,
			searchTerms: []string{
				"find tractor by unit number", "locate tractor", "tractor unit number",
				"available tractors",
			},
			resource: permission.ResourceTractor,
			notes: map[string]string{
				paramStatus:                 noteEquipmentStatus,
				fieldRegistrationExpiryFrom: notePlateLapses,
			},
			references: equipmentReferences(
				"primaryWorkerId", "secondaryWorkerId", "ownerWorkerId",
			),
		},
		{
			name:         toolListTrailers,
			entityPlural: pluralTrailers,
			summary:      trailersSummary,
			searchTerms:  []string{"find trailer by unit number", "available dry vans"},
			resource:     permission.ResourceTrailer,
			notes: map[string]string{
				paramStatus:                 noteEquipmentStatus,
				fieldRegistrationExpiryFrom: notePlateLapses,
				fieldLastInspectionFrom:     noteLastInspection,
			},
			references: equipmentReferences("ownerWorkerId"),
		},
	}
}
