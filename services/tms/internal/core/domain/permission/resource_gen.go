package permission

type Resource string

const (
	// Administration
	ResourceOrganization          Resource = "organization"
	ResourceBusinessUnit          Resource = "business_unit"
	ResourceUser                  Resource = "user"
	ResourceRole                  Resource = "role"
	ResourceAuditLog              Resource = "audit_log"
	ResourceTableConfiguration    Resource = "table_configuration"
	ResourceCustomFieldDefinition Resource = "custom_field_definition"
	ResourceSequenceConfig        Resource = "sequence_config"
	ResourceIntegration           Resource = "integration"
	ResourceAPIKey                Resource = "api_key"
	ResourceDataEntryControl      Resource = "data_entry_control"

	// Equipment
	ResourceEquipmentType         Resource = "equipment_type"
	ResourceEquipmentManufacturer Resource = "equipment_manufacturer"
	ResourceTrailer               Resource = "trailer"
	ResourceTractor               Resource = "tractor"
	ResourceFleetCode             Resource = "fleet_code"

	// Workers
	ResourceWorker    Resource = "worker"
	ResourceWorkerPTO Resource = "worker_pto"

	// Operations
	ResourceShipment              Resource = "shipment"
	ResourceShipmentComment       Resource = "shipment_comment"
	ResourceShipmentMove          Resource = "shipment_move"
	ResourceShipmentStop          Resource = "shipment_stop"
	ResourceShipmentHold          Resource = "shipment_hold"
	ResourceDispatchControl       Resource = "dispatch_control"
	ResourceHoldReason            Resource = "hold_reason"
	ResourceShipmentControl       Resource = "shipment_control"
	ResourceHazmatSegregationRule Resource = "hazmat_segregation_rule"
	ResourceDistanceOverride      Resource = "distance_override"

	// Billing
	ResourceInvoice           Resource = "invoice"
	ResourceBillingQueue      Resource = "billing_queue"
	ResourceAccessorialCharge Resource = "accessorial_charge"
	ResourceChargeType        Resource = "charge_type"
	ResourceRevenueCode       Resource = "revenue_code"
	ResourceFormulaTemplate   Resource = "formula_template"

	// Customers
	ResourceCustomer        Resource = "customer"
	ResourceCustomerContact Resource = "customer_contact"

	// Locations
	ResourceLocation         Resource = "location"
	ResourceLocationCategory Resource = "location_category"

	// Commodities
	ResourceCommodity         Resource = "commodity"
	ResourceHazardousMaterial Resource = "hazardous_material"

	// Accounting
	ResourceAccountingControl        Resource = "accounting_control"
	ResourceBillingControl           Resource = "billing_control"
	ResourceInvoiceAdjustmentControl Resource = "invoice_adjustment_control"
	ResourceAccountType              Resource = "account_type"
	ResourceGeneralLedgerAccount     Resource = "general_ledger_account"
	ResourceDivisionCode             Resource = "division_code"
	ResourceFiscalYear               Resource = "fiscal_year"
	ResourceFiscalPeriod             Resource = "fiscal_period"
	ResourceManualJournal            Resource = "manual_journal"
	ResourceJournalReversal          Resource = "journal_reversal"
	ResourceJournalEntry             Resource = "journal_entry"

	// Compliance
	ResourceQualification          Resource = "qualification"
	ResourceDocumentClassification Resource = "document_classification"
	ResourceDocument               Resource = "document"
	ResourceDocumentType           Resource = "document_type"
	ResourceDocumentControl        Resource = "document_control"
	ResourceDocumentParsingRule    Resource = "document_parsing_rule"

	// Reference Data
	ResourceShipmentType Resource = "shipment_type"
	ResourceServiceType  Resource = "service_type"
	ResourceDelayCode    Resource = "delay_code"
	ResourceReasonCode   Resource = "reason_code"
	ResourceCommentType  Resource = "comment_type"
	ResourceTag          Resource = "tag"

	// Reporting
	ResourceReport    Resource = "report"
	ResourceDashboard Resource = "dashboard"
)

func (r Resource) String() string {
	return string(r)
}
