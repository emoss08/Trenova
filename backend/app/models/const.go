package models

type StatusType string

const (
	Active   StatusType = "A"
	Inactive StatusType = "I"
)

type AcAccountType string

const (
	Ast AcAccountType = "ASSET"
	Lib AcAccountType = "LIABILITY"
	Equ AcAccountType = "EQUITY"
	Rev AcAccountType = "REVENUE"
	Exp AcAccountType = "EXPENSE"
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
	Bank AccountClassification = "BANK"
	Cash AccountClassification = "CASH"
	Ar   AccountClassification = "ACCOUNTS_RECEIVABLE"
	Ap   AccountClassification = "ACCOUNTS_PAYABLE"
	Inv  AccountClassification = "INVENTORY"
	Oca  AccountClassification = "OTHER_CURRENT_ASSET"
	Fa   AccountClassification = "FIXED_ASSET"
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
	TLS         EmailProtocol = "TLS"
	SSL         EmailProtocol = "SSL"
	Unencrypted EmailProtocol = "UNENCRYPTED"
)
