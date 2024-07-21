package constants

// Query parameter values
const (
	QueryParamTrue  = "true"
	QueryParamFalse = "false"
)

// HTTP related constants
const (
	ContentTypeJSON = "application/json"
)

// Validation error reasons
const (
	ReasonMustBePositiveInteger = "Must be a positive integer"
	ReasonInvalidUUID           = "Must be a valid UUID"
	ReasonInvalidDateFormat     = "Must be in RFC3339 format"
)

// Common field names
const (
	FieldLimit    = "limit"
	FieldOffset   = "offset"
	FieldID       = "id"
	FieldFromDate = "fromDate"
	FieldToDate   = "toDate"
)

// Error messages
const (
	ErrInternalServer = "An internal server error occurred"
)

// Permission actions
const (
	ActionView   = "view"
	ActionCreate = "create"
	ActionUpdate = "update"
)

// Entity types
const (
	EntityShipment               = "shipment"
	EntityAccessorialCharge      = "accessorial_charge"
	EntityChargeType             = "charge_type"
	EntityCommentType            = "comment_type"
	EntityCommodity              = "commodity"
	EntityCustomer               = "customer"
	EntityDelayCode              = "delay_code"
	EntityDivisionCode           = "division_code"
	EntityDocumentClassification = "document_classification"
	EntityEquipmentManufacturer  = "equipment_manufacturer"
	EntityEquipmentType          = "equipment_type"
	EntityFleetCode              = "fleet_code"
	EntityGeneralLedgerAccount   = "general_ledger_account"
	EntityHazardousMaterial      = "hazardous_material"
	EntityLocation               = "location"
	EntityLocationCategory       = "location_category"
	EntityOrganization           = "organization"
	EntityQualifierCode          = "qualifier_code"
	EntityReasonCode             = "reason_code"
	EntityRevenueCode            = "revenue_code"
	EntityServiceType            = "service_type"
	EntityShipmentType           = "shipment_type"
	EntityTableChangeAlert       = "table_change_alert"
	EntityTag                    = "tag"
	EntityTractor                = "tractor"
	EntityTrailer                = "trailer"
	EntityUser                   = "user"
	EntityWorker                 = "worker"
)

// Table names
const (
	TableShipment               = "shipments"
	TableAccessorialCharge      = "accessorial_charges"
	TableChargeType             = "charge_types"
	TableCommentType            = "comment_types"
	TableCommodity              = "commodities"
	TableCustomer               = "customers"
	TableDelayCode              = "delay_codes"
	TableDivisionCode           = "division_codes"
	TableDocumentClassification = "document_classifications"
	TableEquipmentManufacturer  = "equipment_manufacturers"
	TableEquipmentType          = "equipment_types"
	TableFleetCode              = "fleet_codes"
	TableGeneralLedgerAccount   = "general_ledger_accounts"
	TableHazardousMaterial      = "hazardous_materials"
	TableLocation               = "locations"
	TableLocationCategory       = "location_categories"
	TableOrganization           = "organizations"
	TableQualifierCode          = "qualifier_codes"
	TableReasonCode             = "reason_codes"
	TableRevenueCode            = "revenue_codes"
	TableServiceType            = "service_types"
	TableShipmentType           = "shipment_types"
	TableTableChangeAlert       = "table_change_alerts"
	TableTag                    = "tags"
	TableTractor                = "tractors"
	TableTrailer                = "trailers"
	TableUser                   = "users"
	TableWorker                 = "workers"
)
