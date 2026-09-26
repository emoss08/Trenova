package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// LateChargeAssessmentRequest asks for late charges as of a date. Preview
// computes what a run would raise and writes nothing.
type LateChargeAssessmentRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	CustomerIDs []pulid.ID            `json:"customerIds"`
	AsOfDate    int64                 `json:"asOfDate"`
	Preview     bool                  `json:"preview"`
}

// LateChargeLine is one period of one invoice.
type LateChargeLine struct {
	InvoiceID             pulid.ID        `json:"invoiceId"`
	InvoiceNumber         string          `json:"invoiceNumber"`
	PeriodIndex           int             `json:"periodIndex"`
	PeriodStart           int64           `json:"periodStart"`
	PeriodEnd             int64           `json:"periodEnd"`
	BasisOpenBalanceMinor int64           `json:"basisOpenBalanceMinor"`
	RatePercent           decimal.Decimal `json:"ratePercent"`
	ChargeMinor           int64           `json:"chargeMinor"`
}

// LateChargeCustomerResult is what the run did, or would do, for one customer.
type LateChargeCustomerResult struct {
	CustomerID       pulid.ID          `json:"customerId"`
	CustomerName     string            `json:"customerName"`
	CurrencyCode     string            `json:"currencyCode"`
	Lines            []*LateChargeLine `json:"lines"`
	TotalChargeMinor int64             `json:"totalChargeMinor"`
	DebitMemoID      pulid.ID          `json:"debitMemoId"`
	DebitMemoNumber  string            `json:"debitMemoNumber"`
	Posted           bool              `json:"posted"`
	Skipped          bool              `json:"skipped"`
	SkipReason       string            `json:"skipReason"`
}

type LateChargeAssessmentResult struct {
	AsOfDate         int64                           `json:"asOfDate"`
	Preview          bool                            `json:"preview"`
	Mode             tenant.LateChargeAssessmentMode `json:"mode"`
	AutoPost         bool                            `json:"autoPost"`
	Customers        []*LateChargeCustomerResult     `json:"customers"`
	MemosCreated     int                             `json:"memosCreated"`
	MemosPosted      int                             `json:"memosPosted"`
	CustomersSkipped int                             `json:"customersSkipped"`
	TotalChargeMinor int64                           `json:"totalChargeMinor"`
}

type LateChargeService interface {
	Assess(
		ctx context.Context,
		req *LateChargeAssessmentRequest,
		actor *RequestActor,
	) (*LateChargeAssessmentResult, error)
	PlanAssess(
		ctx context.Context,
		req *LateChargeAssessmentRequest,
		actor *RequestActor,
	) (*LateChargeAssessmentResult, error)
}
