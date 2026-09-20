package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
)

// The catalog is a curated set, not blanket coverage. There are well over a
// hundred permission resources; publishing a tool per resource would expose
// columns nobody vetted and give the model a menu it cannot reason about. These
// are the entities the assistant is actually asked about.

var (
	statusValues      = []string{"Active", "Inactive"}
	equipmentStatuses = []string{"Available", "OutOfService", "AtMaintenance", "Sold"}
	ownershipTypes    = []string{"CompanyOwned", "Leased", "OwnerOperator"}
	workerTypes       = []string{"Employee", "Contractor"}
	driverTypes       = []string{"Local", "Regional", "OTR", "Team"}
	shipmentStatuses  = []string{
		"New", "PartiallyAssigned", "Assigned", "InTransit", "Delayed",
		"PartiallyCompleted", "ReadyToInvoice", "Completed", "Invoiced", "Canceled",
	}
	billingTransferStates = []string{
		"ReadyForReview", "InReview", "OnHold", "Exception",
		"SentBackToOps", "Approved", "Posted", "Canceled",
	}
	freightTerms     = []string{"Prepaid", "Collect", "ThirdParty"}
	endorsementCodes = []string{"O", "N", "H", "X", "P", "T"}
	complianceStates = []string{"Compliant", "NonCompliant", "Pending"}
	cdlClasses       = []string{"A", "B", "C"}
)

// endorsementNote spells the codes out.
//
// The column stores a single letter, and X is the one that matters: it means
// tanker AND hazmat, so a hazmat question answered with endorsement = 'H' alone
// undercounts the fleet. No model infers that, and one that guesses "Hazmat"
// gets an empty page it will report as nobody holding one.
const endorsementNote = "single letter: O none, N tanker, H hazmat, " +
	"X tanker and hazmat, P passenger, T doubles/triples. " +
	"For hazmat match both H and X"

type workerRow struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	Type              string `json:"type"`
	DriverType        string `json:"driverType"`
	City              string `json:"city,omitempty"`
	FleetCode         string `json:"fleetCode,omitempty"`
	CanBeAssigned     bool   `json:"canBeAssigned"`
	AssignmentBlocked string `json:"assignmentBlocked,omitempty"`
	// The compliance fields travel with the row because they are the reason
	// the row was asked for. A roster filtered on an endorsement that then
	// comes back without it leaves the reader to trust the filter blindly.
	Endorsement       string `json:"endorsement,omitempty"`
	CDLClass          string `json:"cdlClass,omitempty"`
	ComplianceStatus  string `json:"complianceStatus,omitempty"`
	Qualified         *bool  `json:"qualified,omitempty"`
	HazmatExpiry      int64  `json:"hazmatExpiry,omitempty"`
	LicenseExpiry     int64  `json:"licenseExpiry,omitempty"`
	MedicalCardExpiry int64  `json:"medicalCardExpiry,omitempty"`
}

func newListWorkersTool(repo repositories.WorkerRepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_workers",
		entityPlural: "workers",
		summary: "List workers (drivers) narrowed by status, employment type, driver " +
			"type, city or fleet, and by their qualification profile — endorsement, " +
			"CDL class, compliance status, and licence, medical card or hazmat expiry. " +
			"This answers who holds an endorsement and who is qualified to drive. Use " +
			"search_worker when you have a name, and list_expiring_credentials for the " +
			"separately tracked credential documents.",
		resource: permission.ResourceWorker,
		config:   querybuilder.GetFieldConfiguration((*worker.Worker)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "type", Kind: filterEnum, Values: workerTypes, Note: "employee or contractor"},
			{Name: "driverType", Kind: filterEnum, Values: driverTypes},
			{Name: "city", Kind: filterText, Sortable: true},
			{Name: "lastName", Kind: filterText, Sortable: true},
			{Name: "firstName", Kind: filterText},
			{Name: "canBeAssigned", Kind: filterBool, Note: "false means dispatch is blocked"},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
			{
				Name:   "profile.endorsement",
				Kind:   filterEnum,
				Values: endorsementCodes,
				Note:   endorsementNote,
			},
			{Name: "profile.cdlClass", Kind: filterEnum, Values: cdlClasses},
			{
				Name:   "profile.complianceStatus",
				Kind:   filterEnum,
				Values: complianceStates,
			},
			{
				Name: "profile.isQualified",
				Kind: filterBool,
				Note: "the roll-up: false means something is lapsed or missing",
			},
			{
				Name:     "profile.hazmatExpiry",
				Kind:     filterDate,
				Sortable: true,
				Note:     "a current endorsement is one whose expiry is still ahead",
			},
			{Name: "profile.licenseExpiry", Kind: filterDate, Sortable: true},
			{Name: "profile.medicalCardExpiry", Kind: filterDate, Sortable: true},
			{Name: "profile.twicExpiry", Kind: filterDate, Sortable: true},
			{Name: "profile.hireDate", Kind: filterDate, Sortable: true},
			{Name: "profile.terminationDate", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListWorkersRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *worker.Worker) any {
				row := workerRow{
					ID:                item.ID.String(),
					Name:              workerName(item),
					Status:            string(item.Status),
					Type:              string(item.Type),
					DriverType:        string(item.DriverType),
					City:              item.City,
					CanBeAssigned:     item.CanBeAssigned,
					AssignmentBlocked: item.AssignmentBlocked,
				}
				if item.FleetCode != nil {
					row.FleetCode = item.FleetCode.Code
				}
				applyProfile(&row, item.Profile)

				return row
			}), nil
		},
	})
}

type shipmentRow struct {
	ID                 string `json:"id"`
	ProNumber          string `json:"proNumber"`
	BOL                string `json:"bol,omitempty"`
	Status             string `json:"status"`
	Customer           string `json:"customer,omitempty"`
	TotalCharge        string `json:"totalCharge,omitempty"`
	ActualShipDate     int64  `json:"actualShipDate,omitempty"`
	ActualDeliveryDate int64  `json:"actualDeliveryDate,omitempty"`
	BillingStatus      string `json:"billingStatus,omitempty"`
}

func newListShipmentsTool(repo repositories.ShipmentRepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_shipments",
		entityPlural: "shipments",
		summary: "List shipments narrowed by status, billing state, dates or charges. " +
			"This is the tool for anything with a date or a threshold in it — delivered " +
			"yesterday, not yet billed, over a dollar amount. Use search_shipments when " +
			"you are matching text such as a pro number.",
		resource: permission.ResourceShipment,
		config:   querybuilder.GetFieldConfiguration((*shipment.Shipment)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: shipmentStatuses},
			{
				Name:   "billingTransferStatus",
				Kind:   filterEnum,
				Values: billingTransferStates,
				Note:   "where the shipment sits in the billing handoff",
			},
			{Name: "freightTerms", Kind: filterEnum, Values: freightTerms},
			{Name: "proNumber", Kind: filterText, Sortable: true},
			{Name: "bol", Kind: filterText},
			{Name: "actualShipDate", Kind: filterDate, Sortable: true},
			{Name: "actualDeliveryDate", Kind: filterDate, Sortable: true},
			{Name: "billedAt", Kind: filterDate, Sortable: true},
			{Name: "canceledAt", Kind: filterDate},
			{Name: "totalChargeAmount", Kind: filterNumber, Sortable: true},
			{Name: "weight", Kind: filterNumber},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListShipmentsRequest{
				Filter:          opts,
				ShipmentOptions: repositories.ShipmentOptions{IncludeCustomer: true},
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *shipment.Shipment) any {
				row := shipmentRow{
					ID:            item.ID.String(),
					ProNumber:     item.ProNumber,
					BOL:           item.BOL,
					Status:        string(item.Status),
					BillingStatus: string(item.BillingTransferStatus),
				}
				if item.TotalChargeAmount.Valid {
					row.TotalCharge = item.TotalChargeAmount.Decimal.String()
				}
				if item.ActualShipDate != nil {
					row.ActualShipDate = *item.ActualShipDate
				}
				if item.ActualDeliveryDate != nil {
					row.ActualDeliveryDate = *item.ActualDeliveryDate
				}
				if item.Customer != nil {
					row.Customer = item.Customer.Name
				}

				return row
			}), nil
		},
	})
}

type equipmentRow struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	Status             string `json:"status"`
	Make               string `json:"make,omitempty"`
	Model              string `json:"model,omitempty"`
	Year               int    `json:"year,omitempty"`
	LicensePlate       string `json:"licensePlate,omitempty"`
	OwnershipType      string `json:"ownershipType,omitempty"`
	RegistrationExpiry int64  `json:"registrationExpiry,omitempty"`
	LastInspectionDate int64  `json:"lastInspectionDate,omitempty"`
	AssignedTo         string `json:"assignedTo,omitempty"`
}

func equipmentFields() []listField {
	return []listField{
		{Name: "status", Kind: filterEnum, Values: equipmentStatuses},
		{Name: "ownershipType", Kind: filterEnum, Values: ownershipTypes},
		{Name: "code", Kind: filterText, Sortable: true},
		{Name: "make", Kind: filterText},
		{Name: "model", Kind: filterText},
		{Name: "licensePlateNumber", Kind: filterText},
		{Name: "year", Kind: filterNumber, Sortable: true},
		{
			Name:     "registrationExpiry",
			Kind:     filterDate,
			Sortable: true,
			Note:     "when the plate lapses",
		},
		{Name: "leaseEndDate", Kind: filterDate, Sortable: true},
		{Name: "createdAt", Kind: filterDate, Sortable: true},
	}
}

func newListTractorsTool(repo repositories.TractorRepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_tractors",
		entityPlural: "tractors",
		summary: "List tractors (power units) narrowed by status, ownership, or a " +
			"registration or lease date. Use it for questions about which units are out " +
			"of service or coming due for plates.",
		resource: permission.ResourceTractor,
		config:   querybuilder.GetFieldConfiguration((*tractor.Tractor)(nil)),
		fields:   equipmentFields(),
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListTractorsRequest{
				Filter: opts,
				TractorRelationIncludes: repositories.TractorRelationIncludes{
					IncludePrimaryWorker: true,
				},
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *tractor.Tractor) any {
				row := equipmentRow{
					ID:            item.ID.String(),
					Code:          item.Code,
					Status:        string(item.Status),
					Make:          item.Make,
					Model:         item.Model,
					LicensePlate:  item.LicensePlateNumber,
					OwnershipType: string(item.OwnershipType),
				}
				if item.Year != nil {
					row.Year = *item.Year
				}
				if item.RegistrationExpiry != nil {
					row.RegistrationExpiry = *item.RegistrationExpiry
				}
				if item.PrimaryWorker != nil {
					row.AssignedTo = workerName(item.PrimaryWorker)
				}

				return row
			}), nil
		},
	})
}

func newListTrailersTool(repo repositories.TrailerRepository) serviceports.AgentQueryTool {
	fields := append(equipmentFields(), listField{
		Name:     "lastInspectionDate",
		Kind:     filterDate,
		Sortable: true,
		Note:     "the last annual inspection",
	})

	return newListTool(listSpec{
		name:         "list_trailers",
		entityPlural: "trailers",
		summary: "List trailers narrowed by status, ownership, or a registration, " +
			"lease or inspection date. Use it for questions about which trailers are " +
			"out of service or overdue for inspection.",
		resource: permission.ResourceTrailer,
		config:   querybuilder.GetFieldConfiguration((*trailer.Trailer)(nil)),
		fields:   fields,
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListTrailersRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *trailer.Trailer) any {
				row := equipmentRow{
					ID:            item.ID.String(),
					Code:          item.Code,
					Status:        string(item.Status),
					Make:          item.Make,
					Model:         item.Model,
					LicensePlate:  item.LicensePlateNumber,
					OwnershipType: string(item.OwnershipType),
				}
				if item.Year != nil {
					row.Year = *item.Year
				}
				if item.RegistrationExpiry != nil {
					row.RegistrationExpiry = *item.RegistrationExpiry
				}
				if item.LastInspectionDate != nil {
					row.LastInspectionDate = *item.LastInspectionDate
				}

				return row
			}), nil
		},
	})
}

type customerRow struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	City      string `json:"city,omitempty"`
	DOTNumber string `json:"dotNumber,omitempty"`
	MCNumber  string `json:"mcNumber,omitempty"`
}

func newListCustomersTool(repo repositories.CustomerRepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_customers",
		entityPlural: "customers",
		summary: "List customers narrowed by status, code, name or city. Returns their " +
			"ids, which shipment questions can then be filtered by.",
		resource: permission.ResourceCustomer,
		config:   querybuilder.GetFieldConfiguration((*customer.Customer)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "city", Kind: filterText, Sortable: true},
			{Name: "dotNumber", Kind: filterText},
			{Name: "mcNumber", Kind: filterText},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListCustomerRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *customer.Customer) any {
				return customerRow{
					ID:        item.ID.String(),
					Code:      item.Code,
					Name:      item.Name,
					Status:    string(item.Status),
					City:      item.City,
					DOTNumber: item.DOTNumber,
					MCNumber:  item.MCNumber,
				}
			}), nil
		},
	})
}

type locationRow struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Address string `json:"address,omitempty"`
	City    string `json:"city,omitempty"`
}

func newListLocationsTool(repo repositories.LocationRepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_locations",
		entityPlural: "locations",
		summary: "List locations (facilities, terminals, yards) narrowed by status, " +
			"code, name or city.",
		resource: permission.ResourceLocation,
		config:   querybuilder.GetFieldConfiguration((*location.Location)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "city", Kind: filterText, Sortable: true},
			{Name: "postalCode", Kind: filterText},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListLocationRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *location.Location) any {
				return locationRow{
					ID:      item.ID.String(),
					Code:    item.Code,
					Name:    item.Name,
					Status:  string(item.Status),
					Address: item.AddressLine1,
					City:    item.City,
				}
			}), nil
		},
	})
}

// listCatalogSpecs mirrors the registered tools so a test can hold every entry
// against the entity it claims to query. A curated field that stops mapping to a
// column is dropped silently by the query builder, which is the one failure the
// catalog cannot afford.
func listCatalogSpecs() []listSpec {
	return []listSpec{
		specOf(newListWorkersTool(nil)),
		specOf(newListShipmentsTool(nil)),
		specOf(newListTractorsTool(nil)),
		specOf(newListTrailersTool(nil)),
		specOf(newListCustomersTool(nil)),
		specOf(newListLocationsTool(nil)),
		specOf(newListInvoicesTool(nil)),
		specOf(newListCarriersTool(nil)),
		specOf(newListEquipmentTypesTool(nil)),
		specOf(newListFleetCodesTool(nil)),
		specOf(newListServiceTypesTool(nil)),
		specOf(newListShipmentTypesTool(nil)),
		specOf(newListCommoditiesTool(nil)),
		specOf(newListHazardousMaterialsTool(nil)),
		specOf(newListAccessorialChargesTool(nil)),
		specOf(newListDocumentTypesTool(nil)),
		specOf(newListLocationCategoriesTool(nil)),
	}
}

func specOf(tool serviceports.AgentQueryTool) listSpec {
	return tool.(*listTool).spec //nolint:errcheck,forcetypeassert // constructed above
}

// applyProfile copies the qualification roll-up onto the row. A worker without a
// profile is a record mid-onboarding, not an error: the fields stay empty rather
// than reporting a lapsed licence nobody has yet entered.
func applyProfile(row *workerRow, profile *worker.WorkerProfile) {
	if profile == nil {
		return
	}

	row.Endorsement = string(profile.Endorsement)
	row.CDLClass = string(profile.CDLClass)
	row.ComplianceStatus = string(profile.ComplianceStatus)
	row.Qualified = &profile.IsQualified

	if profile.HazmatExpiry != nil {
		row.HazmatExpiry = *profile.HazmatExpiry
	}
	row.LicenseExpiry = profile.LicenseExpiry
	if profile.MedicalCardExpiry != nil {
		row.MedicalCardExpiry = *profile.MedicalCardExpiry
	}
}
