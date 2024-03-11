package models

type StatusType string

const (
	Active   StatusType = "A"
	Inactive StatusType = "I"
)

type AcAccountType string

const (
	AccountTypeAsset     AcAccountType = "ASSET"
	AccountTypeLiability AcAccountType = "LIABILITY"
	AccountTypeEquity    AcAccountType = "EQUITY"
	AccountTypeRevenue   AcAccountType = "REVENUE"
	AccountTypeExpense   AcAccountType = "EXPENSE"
)

type CashFlowType string

const (
	Operating CashFlowType = "OPERATING"
	Investing CashFlowType = "INVESTING"
	Financing CashFlowType = "FINANCING"
)

type AccountSubType string

const (
	CurrentAsset AccountSubType = "CURRENT_ASSET"
	FixedAsset   AccountSubType = "FIXED_ASSET"
	OtherAsset   AccountSubType = "OTHER_ASSET"
	CurrentLib   AccountSubType = "CURRENT_LIABILITY"
	LongTermLib  AccountSubType = "LONG_TERM_LIABILITY"
	Equity       AccountSubType = "EQUITY"
	Revenue      AccountSubType = "REVENUE"
	CostOfGoods  AccountSubType = "COST_OF_GOODS_SOLD"
	Expense      AccountSubType = "EXPENSE"
	OtherIncome  AccountSubType = "OTHER_INCOME"
	OtherExpense AccountSubType = "OTHER_EXPENSE"
)

type AccountClassification string

const (
	AccountClassificationBank AccountClassification = "BANK"
	AccountClassificationCash AccountClassification = "CASH"
	AccountClassificationAR   AccountClassification = "ACCOUNTS_RECEIVABLE"
	AccountClassificationAP   AccountClassification = "ACCOUNTS_PAYABLE"
	AccountClassificationINV  AccountClassification = "INVENTORY"
	AccountClassificationOCA  AccountClassification = "OTHER_CURRENT_ASSET"
	AccountClassificationFA   AccountClassification = "FIXED_ASSET"
)

type TimezoneType string

const (
	Pacific  TimezoneType = "America/Los_Angeles"
	Mountain TimezoneType = "America/Denver"
	Central  TimezoneType = "America/Chicago"
	Eastern  TimezoneType = "America/New_York"
)

type JobFunctionType string

const (
	Manager           JobFunctionType = "MGR"
	ManagementTrainee JobFunctionType = "MT"
	Supervisor        JobFunctionType = "SP"
	Driver            JobFunctionType = "D"
	Billing           JobFunctionType = "B"
	Finance           JobFunctionType = "F"
	Safety            JobFunctionType = "S"
	SysAdmin          JobFunctionType = "SA"
	Admin             JobFunctionType = "A"
)

type AutomaticJournalEntryType string

const (
	OnShipmentBill       AutomaticJournalEntryType = "ON_SHIPMENT_BILL"
	OnReceiptOfPayment   AutomaticJournalEntryType = "ON_RECEIPT_OF_PAYMENT"
	OnExpenseRecognition AutomaticJournalEntryType = "ON_EXPENSE_RECOGNITION"
)

type ThresholdActionType string

const (
	Halt ThresholdActionType = "HALT"
	Warn ThresholdActionType = "WARN"
)

type EmailProtocol string

const (
	TLS                      EmailProtocol = "TLS"
	SSL                      EmailProtocol = "SSL"
	EmailProtocolUnencrypted EmailProtocol = "UNENCRYPTED"
)

type AutoBillShipmentType string

const (
	// AutoBillShipmentDelivery is a constant for the AutoBillShipmentType enum. When a shipment is delivered.
	AutoBillShipmentDelivery AutoBillShipmentType = "D"

	// AutoBillShipmentTransferred is a constant for the AutoBillShipmentType enum. When a shipment is transferred to billing queue.
	AutoBillShipmentTransferred AutoBillShipmentType = "T"

	// AutoBillingMarkedReady is a constant for the AutoBillShipmentType enum. When a shipment is marked ready to bill.
	AutoBillingMarkedReady AutoBillShipmentType = "MR"
)

type ShipmentTransferCriteriaType string

const (
	// ShipmentTransferCriteriaRAndC  is a constant for the ShipmentTransferCriteriaType enum. When a shipment is ready to bill and confirmed.
	ShipmentTransferCriteriaRAndC ShipmentTransferCriteriaType = "RC"

	// ShipmentTransferCriteriaCompleted is a constant for the ShipmentTransferCriteriaType enum. When a shipment is completed.
	ShipmentTransferCriteriaCompleted ShipmentTransferCriteriaType = "C"

	// ShipmentTransferCriteriaReady is a constant for the ShipmentTransferCriteriaType enum. When a shipment is ready to bill.
	ShipmentTransferCriteriaReady ShipmentTransferCriteriaType = "RTB"
)

type DateFormatType string

const (
	MmDdYyyy DateFormatType = "01/02/2006" // MM/DD/YYYY
	DdMmYyyy DateFormatType = "02/01/2006" // DD/MM/YYYY
	YyyyDdMm DateFormatType = "2006/02/01" // YYYY/DD/MM
	YyyyMmDd DateFormatType = "2006/01/02" // YYYY/MM/DD
)

type ServiceIncidentType string

const (
	SiNever             ServiceIncidentType = "N"
	SiPickup            ServiceIncidentType = "P"
	SiDelivery          ServiceIncidentType = "D"
	SiPickupAndDelivery ServiceIncidentType = "PD"
	SiAllExceptPickup   ServiceIncidentType = "AEP"
)

type RouteDistanceUnitType string

const (
	// RduMetric Metric is the same as Kilometers
	RduMetric RouteDistanceUnitType = "M"

	// RduImperial Imperial is the same as Miles
	RduImperial RouteDistanceUnitType = "I"
)

type DistanceMethodType string

const (
	DmGoogle  DistanceMethodType = "G"
	DmTrenova DistanceMethodType = "T"
)
