package accountsreceivable

import "github.com/emoss08/trenova/shared/pulid"

type LedgerEntry struct {
	CustomerID      pulid.ID `json:"customerId"`
	TransactionDate int64    `json:"transactionDate"`
	EventType       string   `json:"eventType"`
	DocumentNumber  string   `json:"documentNumber"`
	SourceObjectID  string   `json:"sourceObjectId"`
	AmountMinor     int64    `json:"amountMinor"`
}

type AgingBucketTotals struct {
	CurrentMinor    int64 `json:"currentMinor"`
	Days1To30Minor  int64 `json:"days1To30Minor"`
	Days31To60Minor int64 `json:"days31To60Minor"`
	Days61To90Minor int64 `json:"days61To90Minor"`
	DaysOver90Minor int64 `json:"daysOver90Minor"`
	TotalOpenMinor  int64 `json:"totalOpenMinor"`
}

type CustomerAgingRow struct {
	CustomerID   pulid.ID          `json:"customerId"`
	CustomerName string            `json:"customerName"`
	Buckets      AgingBucketTotals `json:"buckets"`
}

type AgingSummary struct {
	AsOfDate int64               `json:"asOfDate"`
	Totals   AgingBucketTotals   `json:"totals"`
	Rows     []*CustomerAgingRow `json:"rows"`
}

type OpenItem struct {
	InvoiceID          pulid.ID `json:"invoiceId"`
	CustomerID         pulid.ID `json:"customerId"`
	CustomerName       string   `json:"customerName"`
	InvoiceNumber      string   `json:"invoiceNumber"`
	BillType           string   `json:"billType"`
	InvoiceDate        int64    `json:"invoiceDate"`
	DueDate            int64    `json:"dueDate"`
	CurrencyCode       string   `json:"currencyCode"`
	ShipmentProNumber  string   `json:"shipmentProNumber"`
	ShipmentBOL        string   `json:"shipmentBOL"`
	TotalAmountMinor   int64    `json:"totalAmountMinor"`
	AppliedAmountMinor int64    `json:"appliedAmountMinor"`
	OpenAmountMinor    int64    `json:"openAmountMinor"`
	DaysPastDue        int      `json:"daysPastDue"`
}

type StatementTransaction struct {
	TransactionDate     int64  `json:"transactionDate"`
	EventType           string `json:"eventType"`
	DocumentNumber      string `json:"documentNumber"`
	SourceObjectID      string `json:"sourceObjectId"`
	AmountMinor         int64  `json:"amountMinor"`
	ChargeMinor         int64  `json:"chargeMinor"`
	PaymentMinor        int64  `json:"paymentMinor"`
	RunningBalanceMinor int64  `json:"runningBalanceMinor"`
}

type CustomerStatement struct {
	CustomerID          pulid.ID                `json:"customerId"`
	CustomerName        string                  `json:"customerName"`
	StatementDate       int64                   `json:"statementDate"`
	StartDate           int64                   `json:"startDate"`
	OpeningBalanceMinor int64                   `json:"openingBalanceMinor"`
	TotalChargesMinor   int64                   `json:"totalChargesMinor"`
	TotalPaymentsMinor  int64                   `json:"totalPaymentsMinor"`
	EndingBalanceMinor  int64                   `json:"endingBalanceMinor"`
	Aging               AgingBucketTotals       `json:"aging"`
	Transactions        []*StatementTransaction `json:"transactions"`
	OpenItems           []*OpenItem             `json:"openItems"`
}
