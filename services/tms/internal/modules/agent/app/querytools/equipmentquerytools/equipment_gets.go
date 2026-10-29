package equipmentquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// TractorGetSpec reads one tractor through the Desk's tractor card.
func TractorGetSpec() readspec.Get {
	return readspec.Get{
		Name:     "equipment_get_tractor",
		Entity:   "tractor",
		Resource: permission.ResourceTractor,
		Summary: "Retrieve one tractor by id, with its equipment details, fleet, lease and " +
			"assigned drivers. Use equipment_list_tractors first when you have a unit code.",
		ParamName: "tractorId",
		IDSource:  "from equipment_list_tractors",
	}
}

// TrailerGetSpec reads one trailer through the Desk's trailer card.
func TrailerGetSpec() readspec.Get {
	return readspec.Get{
		Name:     "equipment_get_trailer",
		Entity:   "trailer",
		Resource: permission.ResourceTrailer,
		Summary: "Retrieve one trailer by id, with its equipment details, fleet and lease. " +
			"Use equipment_list_trailers first when you have a unit code.",
		ParamName: "trailerId",
		IDSource:  "from equipment_list_trailers",
	}
}
