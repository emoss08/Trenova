package querytool

const (
	ArgStatuses           = "statuses"
	FieldTargetType       = "targetType"
	AccountingStatusField = "status"
	FieldNumber           = "number"
	FieldCreatedAt        = "createdAt"
	ParamOffset           = "offset"
	ToolListTractors      = "equipment_list_tractors"
	ToolWorkerList        = "worker_list"
	FieldFleetCodeID      = "fleetCodeId"
	FieldEffectiveFrom    = "effectiveFrom"
	FieldEffectiveTo      = "effectiveTo"
	ParamFuelIndexID      = "fuelIndexId"
	FieldWorkerID         = "workerId"
	ParamActiveOnly       = "activeOnly"
	GraphQLListLimit      = 25
	// GraphQLListMost is the server's own page bound (pagination.MaxLimit).
	// A model's read of a page too long for it is cut to the rows that fit,
	// so the bound is what a script reads a page at a time: at 50 its sixty
	// reads stopped at 3,000 rows, short of the 5,000 a script may read.
	GraphQLListMost    = 100
	FilterGroupFilters = "filters"
	// UnpagedMost is the most rows a limit on an unpaged list may ask for: the
	// server's page bound, since such a list has no next page to hand out.
	UnpagedMost                = 100
	RecordIDField              = "id"
	FilterOperator             = "operator"
	FilterValue                = "value"
	MinutesPerHour             = 60
	ListFieldName              = "name"
	ListFieldCode              = "code"
	ListFieldStatus            = "status"
	ToolOrderList              = "order_list"
	ParamAfter                 = "after"
	ToolCarrierList            = "carrier_list"
	FieldPercent               = "percent"
	ParamAdjustmentID          = "adjustmentId"
	ParamReceivableInvoices    = "invoiceIds"
	ToolListEquipmentTypes     = "equipment_list_types"
	ToolListFleetCodes         = "masterdata_list_fleet_codes"
	ToolListLocationCategories = "location_list_categories"
	FieldActive                = "active"
	ParamQuery                 = "query"
	ParamLimit                 = "limit"
	FieldShipmentID            = "shipmentId"
	EntityDriver               = "driver"
	ShipmentListName           = "shipment_list"
	ParamShipmentID            = "shipmentId"
	InputStatus                = "status"
	ActiveStatus               = "Active"
	WfParamOpenOnly            = "openOnly"
	WfParamClosed              = "includeClosed"
)

var (
	StatusValues = []string{"Active", "Inactive"}
)

const (
	ToolCustomerList               = "customer_list"
	ToolLocationList               = "location_list"
	ToolMasterdataListServiceTypes = "masterdata_list_service_types"
	ToolReportGet                  = "report_get"
	ToolReportList                 = "report_list"
)

const ToolListEquipmentManufacturers = "equipment_list_manufacturers"

// listWrapperKeys are the shapes a model reaches for when it has to produce a
// list and nothing told it the shape.
//
// The tool's parameters are a free-form object — they have to be, because every
// report declares its own — so a model filling one in is guessing. Several guess
// {"item": [...]}, which is what an XML-shaped function call looks like once it
// has been through a JSON conversion, and one of them did it five times in a row
// against a report it could otherwise run, narrating a different fix each time
// and sending the identical payload.
//
// The compiler is right to reject it: it serves the reports UI too, where the
// client sends exactly what the definition declares, and loosening it there
// would let a malformed saved report through. This is the agent's boundary, so
// this is where a guess gets straightened out.
var listWrapperKeys = []string{"item", "items", "value", "values", "list"}

// ListDefaults are what a call to a list keyed by an optional worker reads
// with when it names no limit: a page, as the repository tools read.
var ListDefaults = map[string]any{ParamLimit: GraphQLListLimit}
