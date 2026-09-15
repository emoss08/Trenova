package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/latecharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// LateChargeCandidate is an open, overdue invoice of a customer who applies
// late charges, with what the run needs to decide which periods to assess.
type LateChargeCandidate struct {
	CustomerID       pulid.ID        `json:"customerId"`
	CustomerName     string          `json:"customerName"`
	InvoiceID        pulid.ID        `json:"invoiceId"`
	InvoiceNumber    string          `json:"invoiceNumber"`
	CurrencyCode     string          `json:"currencyCode"`
	DueDate          int64           `json:"dueDate"`
	GracePeriodDays  int             `json:"gracePeriodDays"`
	RatePercent      decimal.Decimal `json:"ratePercent"`
	OpenBalanceMinor int64           `json:"openBalanceMinor"`
	// AssessedPeriods are the period indexes already charged on this invoice.
	AssessedPeriods []int `json:"assessedPeriods"`
}

type ListLateChargeCandidatesRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	CustomerIDs []pulid.ID            `json:"customerIds"`
	AsOfDate    int64                 `json:"asOfDate"`
}

type ListLateChargeAssessmentsByInvoiceIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"-"`
}

type LateChargeRepository interface {
	// ListCandidates returns every open, overdue invoice that can take a late
	// charge at AsOfDate: posted invoices and debit memos with a balance, past
	// due plus grace, of customers whose profile applies late charges, that are
	// not disputed and are not themselves late-charge memos.
	ListCandidates(
		ctx context.Context,
		req *ListLateChargeCandidatesRequest,
	) ([]*LateChargeCandidate, error)
	// InsertAssessments writes the rows, skipping any (invoice, period) already
	// assessed, and returns the rows this run actually inserted.
	InsertAssessments(
		ctx context.Context,
		assessments []*latecharge.LateChargeAssessment,
	) ([]*latecharge.LateChargeAssessment, error)
	// SetDebitMemoLines stamps the memo line each assessment was billed on.
	SetDebitMemoLines(ctx context.Context, assessments []*latecharge.LateChargeAssessment) error
	// DeleteByRunKey undoes a run whose memo could not be raised.
	DeleteByRunKey(ctx context.Context, tenantInfo pagination.TenantInfo, runKey string) (int64, error)
	ListBySourceInvoiceIDs(
		ctx context.Context,
		req *ListLateChargeAssessmentsByInvoiceIDsRequest,
	) (map[pulid.ID][]*latecharge.LateChargeAssessment, error)
	ListByDebitMemoIDs(
		ctx context.Context,
		req *ListLateChargeAssessmentsByInvoiceIDsRequest,
	) (map[pulid.ID][]*latecharge.LateChargeAssessment, error)
	// CountBySourceInvoiceID says whether an invoice has been late-charged, which
	// blocks voiding it while its late-charge memos stand.
	CountBySourceInvoiceID(ctx context.Context, tenantInfo pagination.TenantInfo, invoiceID pulid.ID) (int64, error)
	// ListLateChargeTenants returns every tenant whose billing control has late
	// charge assessment switched on.
	ListLateChargeTenants(ctx context.Context, limit int) ([]pagination.TenantInfo, error)
}
