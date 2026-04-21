package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ARLedgerEntry struct {
	CustomerID      pulid.ID `json:"customerId"`
	TransactionDate int64    `json:"transactionDate"`
	EventType       string   `json:"eventType"`
	DocumentNumber  string   `json:"documentNumber"`
	SourceObjectID  string   `json:"sourceObjectId"`
	AmountMinor     int64    `json:"amountMinor"`
}

type ARAgingBucketTotals struct {
	CurrentMinor    int64 `json:"currentMinor"`
	Days1To30Minor  int64 `json:"days1To30Minor"`
	Days31To60Minor int64 `json:"days31To60Minor"`
	Days61To90Minor int64 `json:"days61To90Minor"`
	DaysOver90Minor int64 `json:"daysOver90Minor"`
	TotalOpenMinor  int64 `json:"totalOpenMinor"`
}

type ARCustomerAgingRow struct {
	CustomerID   pulid.ID          `json:"customerId"`
	CustomerName string            `json:"customerName"`
	Buckets      ARAgingBucketTotals `json:"buckets"`
}

type AROpenItem struct {
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

type ListCustomerLedgerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CustomerID pulid.ID              `json:"customerId"`
}

type ListARAgingRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type ListAROpenItemsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CustomerID pulid.ID              `json:"customerId"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type GetARCustomerNameRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CustomerID pulid.ID              `json:"customerId"`
}

type GetARCustomerAgingRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CustomerID pulid.ID              `json:"customerId"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type AccountsReceivableRepository interface {
	ListCustomerLedger(
		ctx context.Context,
		req ListCustomerLedgerRequest,
	) ([]*ARLedgerEntry, error)
	ListARAging(
		ctx context.Context,
		req ListARAgingRequest,
	) ([]*ARCustomerAgingRow, error)
	ListOpenItems(
		ctx context.Context,
		req ListAROpenItemsRequest,
	) ([]*AROpenItem, error)
	GetCustomerName(
		ctx context.Context,
		req GetARCustomerNameRequest,
	) (string, error)
	GetCustomerAging(
		ctx context.Context,
		req GetARCustomerAgingRequest,
	) (*ARCustomerAgingRow, error)
}
