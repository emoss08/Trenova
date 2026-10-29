package partyquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	locationEntity  = "location"
	LocationIDParam = "locationId"
)

// CustomerGetSpec reads one customer through the Desk's customer card.
func CustomerGetSpec() readspec.Get {
	return readspec.Get{
		Name:        "customer_get",
		Entity:      querytool.LabelCustomer,
		SearchTerms: []string{"customer details", "customer profile"},
		Resource:    permission.ResourceCustomer,
		Summary: "Retrieve one customer by id, with their address, authority numbers and " +
			"billing profile: terms, cycle, delivery, credit and the documents each invoice " +
			"needs. Use customer_list first when you have a name or a code rather than an id.",
		ParamName: querytool.ParamCustomerID,
		IDSource:  "from customer_list",
	}
}

// CarrierGetSpec reads one carrier through the Desk's carrier card.
func CarrierGetSpec() readspec.Get {
	return readspec.Get{
		Name:     "carrier_get",
		Entity:   querytool.LabelCarrier,
		Resource: permission.ResourceCarrier,
		Summary: "Retrieve one carrier by id, including its authority, insurance and " +
			"compliance state. Use carrier_list first when you have a name or a code.",
		ParamName: querytool.ParamCarrierID,
		IDSource:  "from carrier_list",
	}
}

// LocationGetSpec reads one location through the Desk's location card.
func LocationGetSpec() readspec.Get {
	return readspec.Get{
		Name:     "location_get",
		Entity:   locationEntity,
		Resource: permission.ResourceLocation,
		Summary: "Retrieve one location by id, with its address, coordinates, geofence, " +
			"timezone and category (appointments, parking, overnight). Use location_list " +
			"first when you have a name, a code or a city.",
		ParamName: LocationIDParam,
		IDSource:  "from location_list",
	}
}
