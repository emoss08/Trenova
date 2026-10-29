package partyquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	customersSummary = "List customers narrowed by status, code, name or city, with their ids. " +
		"Use it to turn a customer's name into an id: shipment_list filters on " +
		"customerId, and a report filters on customer.name."
	locationsSummary = "List locations (facilities, terminals, yards) narrowed by status, " +
		"code, name or city."
	carriersSummary = "List the carriers this organization tenders freight to, narrowed by " +
		"status, compliance, safety rating or authority number. Use it before " +
		"tendering a load to check a carrier is qualified."
	noteQualifiedCarrier = "Only a Qualified carrier should be tendered a load."

	pluralCustomers = "customers"
	pluralLocations = "locations"
	pluralCarriers  = "carriers"

	fieldComplianceStatus   = "complianceStatus"
	fieldLocationCategoryID = "locationCategoryId"
)

// ReadSpecs are the customer, location and carrier lists, read through
// the operation each one's page sends. The repository-backed constructors
// remain for the table composer's filter catalog and for shipment_get's
// related records, which convert with the operations module.
func ReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         querytool.ToolCustomerList,
			EntityPlural: pluralCustomers,
			Summary:      customersSummary,
			Resource:     permission.ResourceCustomer,
		},
		{
			Name:         querytool.ToolLocationList,
			EntityPlural: pluralLocations,
			Summary:      locationsSummary,
			Resource:     permission.ResourceLocation,
			References: map[string]readspec.Reference{fieldLocationCategoryID: {
				Resource: permission.ResourceLocationCategory,
				What:     "The location category",
				Source:   querytool.ToolListLocationCategories,
			}},
		},
		{
			Name:         querytool.ToolCarrierList,
			EntityPlural: pluralCarriers,
			Summary:      carriersSummary,
			Resource:     permission.ResourceCarrier,
			Notes:        map[string]string{fieldComplianceStatus: noteQualifiedCarrier},
		},
	}
}
