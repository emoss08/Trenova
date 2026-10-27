package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	toolListTractors  = "equipment_list_tractors"
	toolListTrailers  = "equipment_list_trailers"
	toolListPositions = "equipment_list_positions"
	toolWorkerList    = "worker_list"

	pluralTractors = "tractors"
	pluralTrailers = "trailers"

	tractorsSummary = "List tractors (power units) narrowed by status, ownership, or a " +
		"registration or lease date. Use it for questions about which units are out " +
		"of service, coming due for plates, or free now (hasInProgressMove false)."
	trailersSummary = "List trailers narrowed by status, ownership, or a registration, " +
		"lease or inspection date, each with its equipment type (dry van, reefer, " +
		"flatbed). Use it for questions about which trailers are out of service, " +
		"overdue for inspection, or free now (hasInProgressMove false)."
	noteEquipmentStatus = "The unit's standing, not whether it is free now: Available means " +
		"it may be dispatched, OutOfService means it cannot, AtMaintenance means it is in " +
		"the shop, and Sold means it has left the fleet. An Available unit may still be on " +
		"a move in progress."
	notePlateLapses    = "When the plate lapses."
	noteLastInspection = "The last annual inspection."

	positionsSummary = "Where every tractor is right now, from the telematics feed: its last " +
		"known position, nearest place, speed and engine state. Each reading names the " +
		"driver and how many minutes old it is. Use it for \"where is truck 104\" or " +
		"\"which trucks are near Dallas\". A reading older than an hour is where the " +
		"truck was, not where it is. Drivers have no position of their own; a driver is " +
		"where the tractor they are logged into is."
	noteMaxAge = "Four hours (14400) when left out."

	paramMaxAgeSeconds     = "maxAgeSeconds"
	paramTractorIDs        = "tractorIds"
	fieldRecordedAt        = "recordedAt"
	fieldAgeMinutes        = "ageMinutes"
	defaultPositionAgeSecs = 4 * 60 * 60

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
		{
			name:         toolListPositions,
			entityPlural: "vehicle positions",
			summary:      positionsSummary,
			searchTerms: []string{
				"truck location", "tractor position", "gps", "telematics", "live position",
			},
			resource: permission.ResourceTractor,
			notes:    map[string]string{paramMaxAgeSeconds: noteMaxAge},
			defaults: map[string]any{paramMaxAgeSeconds: defaultPositionAgeSecs},
			references: map[string]idReference{paramTractorIDs: {
				resource: permission.ResourceTractor,
				what:     "The tractors",
				source:   toolListTractors,
			}},
			shape: ageOfReading,
		},
	}
}

// ageOfReading says how many minutes old each position is, which the screen
// draws from the time and the agent would otherwise work out.
func ageOfReading(row map[string]any, _ map[string]struct{}, clk clock) {
	recorded, ok := typeutils.IntOf(row[fieldRecordedAt])
	if !ok {
		return
	}
	row[fieldAgeMinutes] = max(0, (clk.Now-int64(recorded))/60)
}
