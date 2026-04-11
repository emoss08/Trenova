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
