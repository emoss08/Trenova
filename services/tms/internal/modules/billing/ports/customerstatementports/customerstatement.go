// Package customerstatementports declares how customer statements are
// rendered, emailed and recorded.
package customerstatementports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// MaxHistory bounds how many sent statements one listing returns.
const MaxHistory = 50

type ListStatementCustomersRequest struct {
	TenantInfo pagination.TenantInfo
	Period     customerstatement.Period
}

type ListMonthlySentRequest struct {
	TenantInfo  pagination.TenantInfo
	PeriodEnd   int64
	CustomerIDs []pulid.ID
}

type ListSentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CustomerID pulid.ID              `json:"-"`
}

type Repository interface {
	// ListStatementCustomers lists the customers a statement for the period
	// has something to say to: those owing or owed on its last day, and
	// those with any transaction inside it.
	ListStatementCustomers(
		ctx context.Context,
		req ListStatementCustomersRequest,
	) ([]pulid.ID, error)
	// ListMonthlySent reports which of the customers already received the
	// monthly statement ending at PeriodEnd.
	ListMonthlySent(ctx context.Context, req ListMonthlySentRequest) (map[pulid.ID]bool, error)
	// Insert records a sent statement. A second monthly statement for the
	// same customer and period is ignored and reported as false.
	Insert(ctx context.Context, statement *customerstatement.Statement) (bool, error)
	ListSent(ctx context.Context, req ListSentRequest) ([]*SentStatement, error)
}

// SentStatement is one statement in a customer's history, with who sent it;
// SentByName is blank for a monthly statement.
type SentStatement struct {
	ID                 pulid.ID `json:"id"`
	PeriodStart        int64    `json:"periodStart"`
	PeriodEnd          int64    `json:"periodEnd"`
	EndingBalanceMinor int64    `json:"endingBalanceMinor"`
	PastDueMinor       int64    `json:"pastDueMinor"`
	CurrencyCode       string   `json:"currencyCode"`
	SentTo             string   `json:"sentTo"`
	Monthly            bool     `json:"monthly"`
	SentByName         string   `json:"sentByName"`
	SentAt             int64    `json:"sentAt"`
}

// StatementRequest names the customer and the span a statement covers. A
// zero StartDate runs from the account's first transaction; a zero AsOfDate
// is today.
type StatementRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CustomerID pulid.ID              `json:"-"`
	StartDate  int64                 `json:"startDate" form:"startDate"`
	AsOfDate   int64                 `json:"asOfDate"  form:"asOfDate"`
}

type RenderedStatement struct {
	FileName string
	PDF      []byte
}

type SendRequest struct {
	StatementRequest
	Note string `json:"note"`
}

type SendResult struct {
	SentTo        []string `json:"sentTo"`
	EndingBalance string   `json:"endingBalance"`
	FileName      string   `json:"fileName"`
}

// SweepResult is what one organization's monthly statements did.
type SweepResult struct {
	Customers    int      `json:"customers"`
	Sent         int      `json:"sent"`
	AlreadySent  int      `json:"alreadySent"`
	Failed       int      `json:"failed"`
	NoRecipients []string `json:"noRecipients"`
}

type Service interface {
	Render(ctx context.Context, req *StatementRequest) (*RenderedStatement, error)
	Send(
		ctx context.Context,
		req *SendRequest,
		actor *permissionports.RequestActor,
	) (*SendResult, error)
	SendMonthly(ctx context.Context, control *tenant.BillingControl, now int64) (*SweepResult, error)
	ListSent(ctx context.Context, req ListSentRequest) ([]*SentStatement, error)
}
