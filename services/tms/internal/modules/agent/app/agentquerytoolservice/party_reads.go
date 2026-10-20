package agentquerytoolservice

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

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

	toolCarrierList = "carrier_list"

	pluralCustomers = "customers"
	pluralLocations = "locations"
	pluralCarriers  = "carriers"

	fieldComplianceStatus   = "complianceStatus"
	fieldLocationCategoryID = "locationCategoryId"
)

// partyReadSpecs are the customer, location and carrier lists, read through
// the operation each one's page sends. The repository-backed constructors
// remain for the table composer's filter catalog and for shipment_get's
// related records, which convert with the operations module.
func partyReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolCustomerList,
			entityPlural: pluralCustomers,
			summary:      customersSummary,
			resource:     permission.ResourceCustomer,
		},
		{
			name:         toolLocationList,
			entityPlural: pluralLocations,
			summary:      locationsSummary,
			resource:     permission.ResourceLocation,
			references: map[string]idReference{fieldLocationCategoryID: {
				resource: permission.ResourceLocationCategory,
				what:     "The location category",
				source:   toolListLocationCategories,
			}},
		},
		{
			name:         toolCarrierList,
			entityPlural: pluralCarriers,
			summary:      carriersSummary,
			resource:     permission.ResourceCarrier,
			notes:        map[string]string{fieldComplianceStatus: noteQualifiedCarrier},
		},
	}
}
