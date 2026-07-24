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
	ResourceEmailProfile          Resource = "email_profile"
	ResourceEmailLog              Resource = "email_log"
	ResourceEmailSuppression      Resource = "email_suppression"
	ResourceEDI                   Resource = "edi"
	ResourceAPIKey                Resource = "api_key"
	ResourceDataEntryControl      Resource = "data_entry_control"
	ResourcePlatformCatalog       Resource = "platform_catalog"
	ResourceDatabaseSession       Resource = "database_session"
	ResourceDocumentOperation     Resource = "document_operation"
	ResourceIdentityProvider      Resource = "identity_provider"
	ResourceSCIMDirectory         Resource = "scim_directory"
	ResourceProvisioningAudit     Resource = "provisioning_audit"
	ResourceAccessPolicy          Resource = "access_policy"
	ResourceAuthEvent             Resource = "auth_event"
	ResourceRiskDecision          Resource = "risk_decision"
	ResourceExternalIdentity      Resource = "external_identity"
	ResourceMFAAuthenticator      Resource = "mfa_authenticator"
	ResourceTableChangeAlert      Resource = "table_change_alert"

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
	ResourceOrder                    Resource = "order"
	ResourceShipment                 Resource = "shipment"
	ResourceRecurringShipment        Resource = "recurring_shipment"
	ResourceShipmentComment          Resource = "shipment_comment"
	ResourceShipmentMove             Resource = "shipment_move"
	ResourceShipmentStop             Resource = "shipment_stop"
	ResourceShipmentHold             Resource = "shipment_hold"
	ResourceServiceFailure           Resource = "service_failure"
	ResourceServiceFailureReasonCode Resource = "service_failure_reason_code"
	ResourceDispatchControl          Resource = "dispatch_control"
	ResourceHoldReason               Resource = "hold_reason"
	ResourceShipmentControl          Resource = "shipment_control"
	ResourceHazmatSegregationRule    Resource = "hazmat_segregation_rule"
	ResourceDistanceOverride         Resource = "distance_override"
	ResourceDistanceProfile          Resource = "distance_profile"
	ResourceDistanceControl          Resource = "distance_control"
	ResourceStoredMileage            Resource = "stored_mileage"

	// Billing
	ResourceInvoice              Resource = "invoice"
	ResourceBillingQueue         Resource = "billing_queue"
	ResourceAccessorialCharge    Resource = "accessorial_charge"
	ResourceChargeType           Resource = "charge_type"
	ResourceRevenueCode          Resource = "revenue_code"
	ResourceFormulaTemplate      Resource = "formula_template"
	ResourceRateTable            Resource = "rate_table"
	ResourceFuelSurchargeProgram Resource = "fuel_surcharge_program"

	// Agent
	ResourceAgentRun       Resource = "agent_run"
	ResourceAgentProposal  Resource = "agent_proposal"
	ResourceAgentException Resource = "agent_exception"
	ResourceAgentControl   Resource = "agent_control"

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
	ResourceAccountsReceivable       Resource = "accounts_receivable"
	ResourceBillingControl           Resource = "billing_control"
	ResourceCostingControl           Resource = "costing_control"
	ResourceInvoiceAdjustmentControl Resource = "invoice_adjustment_control"
	ResourceAccountType              Resource = "account_type"
	ResourceGeneralLedgerAccount     Resource = "general_ledger_account"
	ResourceDivisionCode             Resource = "division_code"
	ResourceFiscalYear               Resource = "fiscal_year"
	ResourceFiscalPeriod             Resource = "fiscal_period"
	ResourceManualJournal            Resource = "manual_journal"
	ResourceJournalReversal          Resource = "journal_reversal"
	ResourceJournalEntry             Resource = "journal_entry"
	ResourceCustomerPayment          Resource = "customer_payment"
	ResourceBankReceipt              Resource = "bank_receipt"
	ResourceBankReceiptWorkItem      Resource = "bank_receipt_work_item"
	ResourceAccountingReport         Resource = "accounting_report"

	// Payroll & Settlements
	ResourceDriverPayProfile   Resource = "driver_pay_profile"
	ResourceRecurringDeduction Resource = "recurring_deduction"
	ResourceRecurringEarning   Resource = "recurring_earning"
	ResourcePayCode            Resource = "pay_code"
	ResourcePayAdvance         Resource = "pay_advance"
	ResourceEscrowAccount      Resource = "escrow_account"
	ResourceDriverSettlement   Resource = "driver_settlement"
	ResourceSettlementDispute  Resource = "settlement_dispute"
	ResourceDriverExpense      Resource = "driver_expense"
	ResourceSettlementControl  Resource = "settlement_control"
	ResourceDashControl        Resource = "dash_control"
	ResourceDriverPortal       Resource = "driver_portal"

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
