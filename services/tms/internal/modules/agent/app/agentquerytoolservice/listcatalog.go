package agentquerytoolservice

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
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

const (
	workerProfilePath = "profile."
	workerCityField   = "city"
)

// endorsementNote spells the codes out.
//
// The column stores a single letter, and X is the one that matters: it means
// tanker AND hazmat, so a hazmat question answered with endorsement = 'H' alone
// undercounts the fleet. No model infers that, and one that guesses "Hazmat"
// gets an empty page it will report as nobody holding one.
const endorsementNote = "O none, N tanker, H hazmat, X tanker+hazmat, " +
	"P passenger, T doubles/triples; hazmat is H or X"

type workerRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	DriverType string `json:"driverType"`
	City       string `json:"city,omitempty"`
	FleetCode  string `json:"fleetCode,omitempty"`
	// CanBeAssigned folds the compliance block into the stored flag, because
	// the stored flag alone answers a narrower question than its name does.
	// The column means "no employment-level block"; the domain additionally
	// refuses to dispatch a worker whose credentials are NonCompliant
	// (worker.Standing). A row that reported the column verbatim said a driver
	// whose medical card had lapsed four days earlier was dispatchable.
	CanBeAssigned     bool   `json:"canBeAssigned"`
	AssignmentBlocked string `json:"assignmentBlocked,omitempty"`
	// The compliance fields travel with the row because they are the reason
	// the row was asked for. A roster filtered on an endorsement that then
	// comes back without it leaves the reader to trust the filter blindly.
	Endorsement      string `json:"endorsement,omitempty"`
	CDLClass         string `json:"cdlClass,omitempty"`
	ComplianceStatus string `json:"complianceStatus,omitempty"`
	Qualified        *bool  `json:"qualified,omitempty"`
	// Always present, never omitted: a missing credential date is the finding,
	// and a key that disappears when nothing is on file reads as nothing to
	// worry about.
	HazmatExpiry      optionalDate `json:"hazmatExpiry"`
	LicenseExpiry     optionalDate `json:"licenseExpiry"`
	MedicalCardExpiry optionalDate `json:"medicalCardExpiry"`
}

func newListWorkersTool(
	repo workerports.WorkerRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newListTool(listSpec{
		name:         toolWorkerList,
		entityPlural: "workers",
		summary:      workersSummary,
		searchTerms:  workerSearchTerms,
		queryNote:    workersQueryNote,
		resource:     permission.ResourceWorker,
		config:       listquery.Describe((*worker.Worker)(nil), workerdb.WorkersSchema()),
		fields: []listField{
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: statusValues,
				Note:   "not availability; see canBeAssigned, worker_list_time_off",
			},
			{Name: "type", Kind: filterEnum, Values: workerTypes},
			{Name: "driverType", Kind: filterEnum, Values: driverTypes},
			{Name: "city", Kind: filterText, Sortable: true},
			{Name: "lastName", Kind: filterText, Sortable: true},
			{Name: "firstName", Kind: filterText},
			{Name: "canBeAssigned", Kind: filterBool, Note: "false = dispatch blocked"},
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
				Note: "false = lapsed or missing",
			},
			{Name: "profile.hazmatExpiry", Kind: filterDate},
			{Name: "profile.licenseExpiry", Kind: filterDate},
			{Name: "profile.medicalCardExpiry", Kind: filterDate},
			{Name: "profile.twicExpiry", Kind: filterDate},
		},
		access:        newFieldAccess(permissions),
		gatedField:    workerGatedField,
		pagesByCursor: true,
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			cursor pagination.CursorInfo,
			gate *fieldGate,
		) (listPage, error) {
			result, err := repo.List(
				ctx,
				&workerports.ListWorkersRequest{Filter: opts, Cursor: cursor},
			)
			if err != nil {
				return listPage{}, err
			}
			showCity := gate.show(workerCityField, workerCityField)

			listed, err := cursorListPage(result, func(item *worker.Worker) any {
				return toGatedWorkerRow(item, showCity)
			})
			if err != nil {
				return listPage{}, err
			}
			if len(opts.Sort) == 0 {
				listed.rows = rankWorkersByName(opts.Query, result.Items, listed.rows)
			}

			return listed, nil
		},
	})
}

func rankWorkersByName(query string, items []*worker.Worker, rows []any) []any {
	if strings.TrimSpace(query) == "" || len(items) < 2 || len(items) != len(rows) {
		return rows
	}

	type scored struct {
		row   any
		score float64
	}
	ranked := make([]scored, 0, len(items))
	for idx, item := range items {
		ranked = append(ranked, scored{row: rows[idx], score: workerNameScore(query, item)})
	}
	slices.SortStableFunc(ranked, func(a, b scored) int {
		return cmp.Compare(b.score, a.score)
	})
	out := make([]any, 0, len(ranked))
	for idx := range ranked {
		out = append(out, ranked[idx].row)
	}

	return out
}

func workerNameScore(query string, item *worker.Worker) float64 {
	return namesScore(query, item.FirstName, item.LastName)
}

// namesScore is how closely a query matches a person's name, either way
// round.
func namesScore(query, first, last string) float64 {
	return max(
		stringutils.NameSimilarity(query, strings.TrimSpace(first+" "+last)),
		stringutils.NameSimilarity(query, last+" "+first),
	)
}

func workerGatedField(field string) string {
	return strings.TrimPrefix(field, workerProfilePath)
}

type shipmentRow struct {
	ID                 string            `json:"id"`
	ProNumber          string            `json:"proNumber"`
	BOL                string            `json:"bol,omitempty"`
	Status             string            `json:"status"`
	Customer           string            `json:"customer,omitempty"`
	Pickup             string            `json:"pickup,omitempty"`
	Delivery           string            `json:"delivery,omitempty"`
	Drivers            string            `json:"drivers,omitempty"`
	Carrying           string            `json:"carrying,omitempty"`
	TotalCharge        string            `json:"totalCharge,omitempty"`
	ActualShipDate     optionalDate      `json:"actualShipDate"`
	ActualDeliveryDate optionalDate      `json:"actualDeliveryDate"`
	BillingStatus      string            `json:"billingStatus,omitempty"`
	ActiveHolds        []shipmentRowHold `json:"activeHolds,omitempty"`
}

func shipmentListSpec(repo shipmentports.ShipmentRepository) listSpec {
	return listSpec{
		name:         shipmentListName,
		row:          shipmentRow{},
		entityPlural: "shipments",
		summary: "Shipments by status, billing state, dates or charges, such as picking up " +
			"today.",
		resource: permission.ResourceShipment,
		config:   listquery.Describe((*shipment.Shipment)(nil), shipmentdb.ShipmentsSchema()),
		fields: []listField{
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: shipmentStatuses,
				Note: "rolling is InTransit and Delayed, one in filter; no Delivered: a " +
					"delivered load is Completed, ReadyToInvoice, then Invoiced",
			},
			{
				Name:   "billingTransferStatus",
				Kind:   filterEnum,
				Values: billingTransferStates,
				Note:   "isnull means never transferred",
			},
			{Name: "freightTerms", Kind: filterEnum, Values: freightTerms},
			{
				Name: "customerId",
				Kind: filterText,
				Note: "from customer_list, never guessed",
			},
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
		pagesByCursor: true,
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, cursor pagination.CursorInfo) (listPage, error) {
			result, err := repo.List(ctx, &shipmentports.ListShipmentsRequest{
				Filter: opts,
				Cursor: cursor,
				ShipmentOptions: shipmentports.ShipmentOptions{
					IncludeCustomer:    true,
					IncludeRoute:       true,
					IncludeCommodities: true,
				},
			})
			if err != nil {
				return listPage{}, err
			}

			return cursorListPage(result, func(item *shipment.Shipment) any {
				return toShipmentRow(item)
			})
		},
		count: func(ctx context.Context, opts *pagination.QueryOptions) (int, bool, error) {
			return countShipments(ctx, repo, opts, shipmentports.ShipmentOptions{})
		},
	}
}

var errNoTotal = errors.New("the shipment count was not returned")

func countShipments(
	ctx context.Context,
	repo shipmentports.ShipmentRepository,
	opts *pagination.QueryOptions,
	options shipmentports.ShipmentOptions,
) (int, bool, error) {
	probe := *opts
	probe.Pagination = pagination.Info{Limit: 1}
	probe.Sort = nil

	result, err := repo.List(ctx, &shipmentports.ListShipmentsRequest{
		Filter:          &probe,
		Cursor:          pagination.CursorInfo{Limit: 1, IncludeTotalCount: true},
		ShipmentOptions: options,
	})
	if err != nil {
		return 0, false, err
	}
	if result.TotalCount == nil {
		return 0, false, errNoTotal
	}

	return *result.TotalCount, result.TotalCountExact, nil
}

// toShipmentRow is the one shipment projection, for the same reasons.
func toShipmentRow(item *shipment.Shipment) shipmentRow {
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
	row.ActualShipDate = expectedDate(typeutils.ValueOrZero(item.ActualShipDate), "not shipped yet")
	row.ActualDeliveryDate = expectedDate(
		typeutils.ValueOrZero(item.ActualDeliveryDate), "not delivered yet")
	if item.Customer != nil {
		row.Customer = item.Customer.Name
	}
	route := routeOf(item)
	row.Pickup = route.pickup
	row.Delivery = route.delivery
	row.Drivers = route.coverage
	row.Carrying = carriedCommodities(item.Commodities)

	return row
}

func carriedCommodities(commodities []*shipment.ShipmentCommodity) string {
	names := make([]string, 0, len(commodities))
	for _, line := range commodities {
		if line == nil || line.Commodity == nil {
			continue
		}
		if name := strings.TrimSpace(line.Commodity.Name); name != "" &&
			!slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	return strings.Join(names, ", ")
}

type equipmentRow struct {
	ID                 string       `json:"id"`
	Code               string       `json:"code"`
	Status             string       `json:"status"`
	Make               string       `json:"make,omitempty"`
	Model              string       `json:"model,omitempty"`
	Year               int          `json:"year,omitempty"`
	LicensePlate       string       `json:"licensePlate,omitempty"`
	OwnershipType      string       `json:"ownershipType,omitempty"`
	RegistrationExpiry optionalDate `json:"registrationExpiry"`
	LastInspectionDate optionalDate `json:"lastInspectionDate"`
	EquipmentType      string       `json:"equipmentType,omitempty"`
	AssignedTo         string       `json:"assignedTo,omitempty"`
}

func equipmentTypeName(kind *equipmenttype.EquipmentType) string {
	switch {
	case kind == nil:
		return ""
	case kind.Description == "":
		return kind.Code
	default:
		return kind.Code + " — " + kind.Description
	}
}

func equipmentFields() []listField {
	return []listField{
		{
			Name:   "status",
			Kind:   filterEnum,
			Values: equipmentStatuses,
			Note:   noteEquipmentStatus,
		},
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
			Note:     notePlateLapses,
		},
		{Name: "leaseEndDate", Kind: filterDate, Sortable: true},
		{Name: "createdAt", Kind: filterDate, Sortable: true},
	}
}

func newListTractorsTool(
	repo tractorports.TractorRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newListTool(listSpec{
		name:         toolListTractors,
		row:          equipmentRow{},
		entityPlural: pluralTractors,
		summary:      tractorsSummary,
		searchTerms: []string{
			"find tractor by unit number", "locate tractor", "tractor unit number",
			"available tractors",
		},
		resource:      permission.ResourceTractor,
		config:        listquery.Describe((*tractor.Tractor)(nil), tractordb.TractorsSchema()),
		fields:        equipmentFields(),
		access:        newFieldAccess(permissions),
		pagesByCursor: true,
		fetchJoined: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			cursor pagination.CursorInfo,
			mayRead func(permission.Resource) bool,
		) (listPage, error) {
			nameDrivers := mayRead(permission.ResourceWorker)
			result, err := repo.List(ctx, &tractorports.ListTractorsRequest{
				Filter: opts,
				Cursor: cursor,
				TractorRelationIncludes: tractorports.TractorRelationIncludes{
					IncludePrimaryWorker: nameDrivers,
					IncludeEquipmentType: true,
				},
			})
			if err != nil {
				return listPage{}, err
			}

			return cursorListPage(result, func(item *tractor.Tractor) any {
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
				row.RegistrationExpiry = pointerDate(item.RegistrationExpiry)
				row.EquipmentType = equipmentTypeName(item.EquipmentType)
				if nameDrivers {
					row.AssignedTo = workerName(item.PrimaryWorker)
				}

				return row
			})
		},
	})
}

func newListTrailersTool(repo trailerports.TrailerRepository) agenttoolports.AgentQueryTool {
	fields := append(equipmentFields(), listField{
		Name:     "lastInspectionDate",
		Kind:     filterDate,
		Sortable: true,
		Note:     noteLastInspection,
	})

	return newListTool(listSpec{
		name:         toolListTrailers,
		row:          equipmentRow{},
		entityPlural: pluralTrailers,
		summary:      trailersSummary,
		searchTerms: []string{
			"find trailer by unit number", "available dry vans",
		},
		resource:      permission.ResourceTrailer,
		config:        listquery.Describe((*trailer.Trailer)(nil), trailerdb.TrailersSchema()),
		fields:        fields,
		pagesByCursor: true,
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, cursor pagination.CursorInfo) (listPage, error) {
			result, err := repo.List(ctx, &trailerports.ListTrailersRequest{
				Filter: opts,
				Cursor: cursor,
				TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
					IncludeEquipmentType: true,
				},
			})
			if err != nil {
				return listPage{}, err
			}

			return cursorListPage(result, func(item *trailer.Trailer) any {
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
				row.RegistrationExpiry = pointerDate(item.RegistrationExpiry)
				row.LastInspectionDate = pointerDate(item.LastInspectionDate)
				row.EquipmentType = equipmentTypeName(item.EquipmentType)

				return row
			})
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

func newListCustomersTool(repo customerports.CustomerRepository) agenttoolports.AgentQueryTool {
	return newListTool(listSpec{
		name:         toolCustomerList,
		row:          customerRow{},
		entityPlural: pluralCustomers,
		summary:      customersSummary,
		resource:     permission.ResourceCustomer,
		config:       listquery.Describe((*customer.Customer)(nil), customerdb.CustomersSchema()),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "city", Kind: filterText, Sortable: true},
			{Name: "dotNumber", Kind: filterText},
			{Name: "mcNumber", Kind: filterText},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listPage, error) {
			result, err := repo.List(ctx, &customerports.ListCustomerRequest{Filter: opts})
			if err != nil {
				return listPage{}, err
			}

			return offsetResultPage(result, func(item *customer.Customer) any {
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

func newListLocationsTool(repo locationports.LocationRepository) agenttoolports.AgentQueryTool {
	return newListTool(listSpec{
		name:         toolLocationList,
		row:          locationRow{},
		entityPlural: pluralLocations,
		summary:      locationsSummary,
		resource:     permission.ResourceLocation,
		config:       listquery.Describe((*location.Location)(nil), locationdb.LocationsSchema()),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: statusValues},
			{Name: "code", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "city", Kind: filterText, Sortable: true},
			{Name: "postalCode", Kind: filterText},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions, _ pagination.CursorInfo) (listPage, error) {
			result, err := repo.List(ctx, &locationports.ListLocationRequest{Filter: opts})
			if err != nil {
				return listPage{}, err
			}

			return offsetResultPage(result, func(item *location.Location) any {
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
		specOf(newListWorkersTool(nil, nil)),
		shipmentListSpec(nil),
		specOf(newListTractorsTool(nil, nil)),
		specOf(newListTrailersTool(nil)),
		specOf(newListCustomersTool(nil)),
		specOf(newListLocationsTool(nil)),
		specOf(newListInvoicesTool(nil, nil)),
		specOf(newListBillingQueueItemsTool(nil, nil)),
		specOf(newListCarriersTool(nil)),
		specOf(newListEquipmentTypesTool(nil)),
		specOf(newListFleetCodesTool(nil)),
		specOf(newListServiceTypesTool(nil)),
		specOf(newListShipmentTypesTool(nil)),
		specOf(newListCommoditiesTool(nil)),
		specOf(newListCarrierCapacityPostingsTool(nil)),
		specOf(newListHazardousMaterialsTool(nil)),
		specOf(newListAccessorialChargesTool(nil)),
		specOf(newListDocumentTypesTool(nil)),
		specOf(newListLocationCategoriesTool(nil)),
		specOf(newListHoldReasonsTool(nil)),
		specOf(newListGLAccountsTool(nil, nil)),
		specOf(newListFiscalPeriodsTool(nil, nil)),
		specOf(newListOrdersTool(nil, nil)),
		specOf(newListRecurringShipmentsTool(nil)),
		specOf(newListFormulaTemplatesTool(nil)),
		specOf(newListEquipmentManufacturersTool(nil)),
	}
}

func specOf(tool agenttoolports.AgentQueryTool) listSpec {
	return tool.(*listTool).spec //nolint:errcheck,forcetypeassert // constructed above
}

// applyProfile copies the qualification roll-up onto the row. A worker without a
// profile is a record mid-onboarding, not an error: the fields stay empty rather
// than reporting a lapsed licence nobody has yet entered.
// toWorkerRow is the one worker projection. The worker name search used to
// return the stored entity instead, which cost six times the bytes for the
// same answer — eight drivers overran the tool-result cap and were truncated
// mid-record — and reported an unrecorded credential as a bare null, which is
// the reading that caused the incident these rows exist to prevent.
func toWorkerRow(item *worker.Worker) workerRow {
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
}

func toGatedWorkerRow(item *worker.Worker, showCity bool) workerRow {
	row := toWorkerRow(item)
	if !showCity {
		row.City = ""
	}

	return row
}

// dispatchable folds a worker's credential compliance into the stored
// dispatch flag, as worker.Standing does: a NonCompliant worker cannot be
// assigned whatever the column says.
func dispatchable(
	canBeAssigned bool,
	blocked string,
	compliance worker.ComplianceStatus,
) (assignable bool, reason string) {
	if compliance != worker.ComplianceStatusNonCompliant {
		return canBeAssigned, blocked
	}
	if blocked == "" {
		blocked = "Credentials are not compliant"
	}

	return false, blocked
}

func applyProfile(row *workerRow, profile *worker.WorkerProfile) {
	if profile == nil {
		return
	}

	row.Endorsement = string(profile.Endorsement)
	row.CDLClass = string(profile.CDLClass)
	row.ComplianceStatus = string(profile.ComplianceStatus)
	row.Qualified = &profile.IsQualified

	// The same rule worker.Standing applies, at the only point this projection
	// can apply it. Standing also weighs training, safety and checklists, which
	// a list row does not load — so this can still be too generous, never too
	// strict, and a caller deciding whether to dispatch should open the record.
	row.CanBeAssigned, row.AssignmentBlocked = dispatchable(
		row.CanBeAssigned, row.AssignmentBlocked, profile.ComplianceStatus,
	)

	row.HazmatExpiry = pointerDate(profile.HazmatExpiry)
	row.LicenseExpiry = recordedDate(profile.LicenseExpiry)
	row.MedicalCardExpiry = pointerDate(profile.MedicalCardExpiry)
}

// FilterCatalog is the vocabulary every catalogued resource answers to.
//
// It is built from the same specs the list tools are, with nil repositories,
// because what a caller may narrow an entity by is a property of the entity
// and not of the connection. That is what lets the Ask input on a data table
// compile "shipments in transit" into the filters shipment_list would have
// built from the same words.
func FilterCatalog() *filtercatalog.Catalog {
	specs := listCatalogSpecs()
	resources := make([]filtercatalog.Resource, 0, len(specs))
	for _, spec := range specs {
		resources = append(resources, catalogResource(spec))
	}

	return filtercatalog.New(resources...)
}
