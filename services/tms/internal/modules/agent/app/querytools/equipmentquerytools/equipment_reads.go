package equipmentquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	toolListTrailers  = "equipment_list_trailers"
	toolListPositions = "equipment_list_positions"

	pluralTractors = "tractors"
	pluralTrailers = "trailers"

	tractorsSummary = "List tractors (power units) narrowed by status, ownership, or a " +
		"registration or lease date. Use it for questions about which units are out " +
		"of service, coming due for plates, or free now (hasInProgressMove false). " +
		unitNumberNote
	trailersSummary = "List trailers narrowed by status, ownership, or a registration, " +
		"lease or inspection date, each with its equipment type (dry van, reefer, " +
		"flatbed). Use it for questions about which trailers are out of service, " +
		"overdue for inspection, or free now (hasInProgressMove false). " + unitNumberNote
	unitNumberNote      = "Name a unit by its unitNumber, never by its plate."
	fieldUnitCode       = "code"
	fieldUnitNumber     = "unitNumber"
	noteEquipmentStatus = "The unit's standing, not whether it is free now: Available means " +
		"it may be dispatched, OutOfService means it cannot, AtMaintenance means it is in " +
		"the shop, and Sold means it has left the fleet. An Available unit may still be on " +
		"a move in progress, and may still be booked for its next one: when the person " +
		"names it, propose it and say what it is still finishing."
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
	fieldLastKnownLocationID     = "lastKnownLocationId"
)

// equipmentReferences are the ids a tractor or trailer filters by, and the
// tool that supplies each.
func equipmentReferences(workers ...string) map[string]readspec.Reference {
	refs := map[string]readspec.Reference{
		fieldEquipmentTypeID: {
			Resource: permission.ResourceEquipmentType,
			What:     "The equipment type",
			Source:   querytool.ToolListEquipmentTypes,
		},
		fieldEquipmentManufacturerID: {
			Resource: permission.ResourceEquipmentManufacturer,
			What:     "The manufacturer",
			Source:   querytool.ToolListEquipmentManufacturers,
		},
		querytool.FieldFleetCodeID: {
			Resource: permission.ResourceFleetCode,
			What:     "The fleet code",
			Source:   querytool.ToolListFleetCodes,
		},
		fieldLastKnownLocationID: {
			Resource: permission.ResourceLocation,
			What:     "The location it was last seen at",
			Source:   querytool.ToolLocationList,
		},
	}
	for _, field := range workers {
		refs[field] = readspec.WorkerReference
	}

	return refs
}

// ReadSpecs are the tractor and trailer lists, read through the
// operation each one's page sends. A tractor's drivers are workers, so a
// reader who cannot read workers sees only their ids, as before.
func ReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         querytool.ToolListTractors,
			EntityPlural: pluralTractors,
			Summary:      tractorsSummary,
			SearchTerms: []string{
				"find tractor by unit number", "locate tractor", "tractor unit number",
				"available tractors",
			},
			Resource: permission.ResourceTractor,
			Names:    map[string]string{fieldUnitCode: fieldUnitNumber},
			Presents: map[string]string{fieldUnitCode: fieldUnitNumber},
			Notes: map[string]string{
				querytool.ParamStatus:       noteEquipmentStatus,
				fieldRegistrationExpiryFrom: notePlateLapses,
			},
			References: equipmentReferences(
				"primaryWorkerId", "secondaryWorkerId", "ownerWorkerId",
			),
		},
		{
			Name:         toolListTrailers,
			EntityPlural: pluralTrailers,
			Summary:      trailersSummary,
			SearchTerms:  []string{"find trailer by unit number", "available dry vans"},
			Resource:     permission.ResourceTrailer,
			Names:        map[string]string{fieldUnitCode: fieldUnitNumber},
			Presents:     map[string]string{fieldUnitCode: fieldUnitNumber},
			Notes: map[string]string{
				querytool.ParamStatus:       noteEquipmentStatus,
				fieldRegistrationExpiryFrom: notePlateLapses,
				fieldLastInspectionFrom:     noteLastInspection,
			},
			References: equipmentReferences("ownerWorkerId"),
		},
		{
			Name:         toolListPositions,
			EntityPlural: "vehicle positions",
			Summary:      positionsSummary,
			SearchTerms: []string{
				"truck location", "tractor position", "gps", "telematics", "live position",
			},
			Resource: permission.ResourceTractor,
			Notes:    map[string]string{paramMaxAgeSeconds: noteMaxAge},
			Defaults: map[string]any{paramMaxAgeSeconds: defaultPositionAgeSecs},
			References: map[string]readspec.Reference{paramTractorIDs: {
				Resource: permission.ResourceTractor,
				What:     "The tractors",
				Source:   querytool.ToolListTractors,
			}},
			Shape: ageOfReading,
		},
	}
}

// ageOfReading says how many minutes old each position is, which the screen
// draws from the time and the agent would otherwise work out.
func ageOfReading(row map[string]any, _ map[string]struct{}, clk filtercatalog.Clock) {
	recorded, ok := typeutils.IntOf(row[fieldRecordedAt])
	if !ok {
		return
	}
	row[fieldAgeMinutes] = max(0, (clk.Now-int64(recorded))/60)
}
