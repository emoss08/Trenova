package collectionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// CollectorScope narrows the worklist by who is chasing the account.
type CollectorScope string

const (
	CollectorScopeAll        = CollectorScope("All")
	CollectorScopeMine       = CollectorScope("Mine")
	CollectorScopeUnassigned = CollectorScope("Unassigned")
)

func (s CollectorScope) IsValid() bool {
	switch s {
	case CollectorScopeAll, CollectorScopeMine, CollectorScopeUnassigned:
		return true
	default:
		return false
	}
}

// WorklistRequest asks for the customers collections should be working.
// DueOnly keeps the accounts whose follow-up has come round, or that nobody
// has scheduled a follow-up for yet.
type WorklistRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Scope      CollectorScope        `json:"scope"   form:"scope"`
	DueOnly    bool                  `json:"dueOnly" form:"dueOnly"`
	Search     string                `json:"search"  form:"search"`
	Limit      int                   `json:"limit"   form:"limit"`
	Offset     int                   `json:"offset"  form:"offset"`
	AsOf       int64                 `json:"-"`
}

// Promise is the promise to pay standing on an account and how it is going.
type Promise struct {
	ActivityID     pulid.ID                 `json:"activityId"`
	AmountMinor    int64                    `json:"amountMinor"`
	CurrencyCode   string                   `json:"currencyCode"`
	PromisedOn     int64                    `json:"promisedOn"`
	MadeAt         int64                    `json:"madeAt"`
	MadeByName     string                   `json:"madeByName"`
	PaidSinceMinor int64                    `json:"paidSinceMinor"`
	Status         collection.PromiseStatus `json:"status"`
}

// WorklistRow is one customer to chase.
type WorklistRow struct {
	CustomerID           pulid.ID                `json:"customerId"`
	CustomerName         string                  `json:"customerName"`
	CustomerCode         string                  `json:"customerCode"`
	CreditStatus         string                  `json:"creditStatus"`
	CurrencyCode         string                  `json:"currencyCode"`
	TotalOpenMinor       int64                   `json:"totalOpenMinor"`
	OverdueMinor         int64                   `json:"overdueMinor"`
	OldestDaysPastDue    int                     `json:"oldestDaysPastDue"`
	OpenInvoiceCount     int                     `json:"openInvoiceCount"`
	DisputedInvoiceCount int                     `json:"disputedInvoiceCount"`
	CollectorID          pulid.ID                `json:"collectorId"`
	CollectorName        string                  `json:"collectorName"`
	NextFollowUpOn       *int64                  `json:"nextFollowUpOn"`
	FollowUpDue          bool                    `json:"followUpDue"`
	LastActivityAt       *int64                  `json:"lastActivityAt"`
	LastActivityKind     collection.ActivityKind `json:"lastActivityKind"`
	LastActivitySummary  string                  `json:"lastActivitySummary"`
	Promise              *Promise                `json:"promise"`
	Priority             collection.Priority     `json:"priority"`
}

// WorklistSummary totals the whole worklist, not only the page.
type WorklistSummary struct {
	Accounts int `json:"accounts"`
	// Overdue is the overdue total per currency, largest first: receivables
	// billed in different currencies are never added together.
	Overdue        []CurrencyTotal `json:"overdue"`
	FollowUpsDue   int             `json:"followUpsDue"`
	BrokenPromises int             `json:"brokenPromises"`
	Unassigned     int             `json:"unassigned"`
}

type CurrencyTotal struct {
	CurrencyCode string `json:"currencyCode"`
	AmountMinor  int64  `json:"amountMinor"`
}

type WorklistPage struct {
	Rows    []*WorklistRow  `json:"rows"`
	Total   int             `json:"total"`
	Summary WorklistSummary `json:"summary"`
}

type GetAccountRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CustomerID pulid.ID              `json:"-"`
}

// ActivityCursor pages a customer's history backwards from an entry.
type ActivityCursor struct {
	CreatedAt int64    `json:"createdAt" form:"beforeCreatedAt"`
	ID        pulid.ID `json:"id"        form:"beforeId"`
}

type ListActivitiesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CustomerID pulid.ID              `json:"-"`
	Before     *ActivityCursor       `json:"-"`
	Limit      int                   `json:"-"`
}

// ActivityView is a history entry with the names a person reads.
type ActivityView struct {
	*collection.Activity
	CreatedByName string `json:"createdByName"`
	CollectorName string `json:"collectorName"`
	InvoiceNumber string `json:"invoiceNumber"`
}

// AccountView is everything the collections panel shows for one customer.
type AccountView struct {
	Row        *WorklistRow        `json:"row"`
	Account    *collection.Account `json:"account"`
	Activities []*ActivityView     `json:"activities"`
	HasMore    bool                `json:"hasMore"`
}

type Repository interface {
	GetAccountForUpdate(ctx context.Context, req GetAccountRequest) (*collection.Account, error)
	SaveAccount(ctx context.Context, account *collection.Account) (*collection.Account, error)
	InsertActivity(ctx context.Context, activity *collection.Activity) error
	ListActivities(ctx context.Context, req ListActivitiesRequest) ([]*ActivityView, error)
	ListWorklist(ctx context.Context, req *WorklistRequest) (*WorklistPage, error)
	GetWorklistRow(ctx context.Context, req GetAccountRequest, asOf int64) (*WorklistRow, error)
	ListAwaitingPromise(ctx context.Context, req AwaitingPromiseRequest) (map[pulid.ID]bool, error)
}

// AwaitingPromiseRequest asks which of the customers stand behind a promise to
// pay whose date has not passed and which is not yet paid in full: reminding
// them now would chase money they have already said is coming.
type AwaitingPromiseRequest struct {
	TenantInfo  pagination.TenantInfo
	CustomerIDs []pulid.ID
	AsOf        int64
}

// LogActivityRequest records a contact with the customer. A promise to pay
// takes an amount and a date; FollowUpOn moves the account's next follow-up,
// which a promise otherwise sets to the day after the promised date.
type LogActivityRequest struct {
	TenantInfo     pagination.TenantInfo   `json:"-"`
	CustomerID     pulid.ID                `json:"-"`
	Kind           collection.ActivityKind `json:"kind"`
	Summary        string                  `json:"summary"`
	ContactName    string                  `json:"contactName"`
	InvoiceID      pulid.ID                `json:"invoiceId"`
	PromisedAmount decimal.NullDecimal     `json:"promisedAmount"`
	PromisedOn     *int64                  `json:"promisedOn"`
	FollowUpOn     *int64                  `json:"followUpOn"`
}

// AssignCollectorRequest gives the account to a collector, or to nobody when
// CollectorID is nil.
type AssignCollectorRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	CustomerID  pulid.ID              `json:"-"`
	CollectorID pulid.ID              `json:"collectorId"`
}

// ScheduleFollowUpRequest sets when the account is next chased, or clears it.
type ScheduleFollowUpRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CustomerID pulid.ID              `json:"-"`
	FollowUpOn *int64                `json:"followUpOn"`
	Note       string                `json:"note"`
}

type Service interface {
	Worklist(ctx context.Context, req *WorklistRequest) (*WorklistPage, error)
	GetAccount(
		ctx context.Context,
		req GetAccountRequest,
		before *ActivityCursor,
	) (*AccountView, error)
	LogActivity(
		ctx context.Context,
		req *LogActivityRequest,
		actor *permissionports.RequestActor,
	) (*AccountView, error)
	AssignCollector(
		ctx context.Context,
		req *AssignCollectorRequest,
		actor *permissionports.RequestActor,
	) (*AccountView, error)
	ScheduleFollowUp(
		ctx context.Context,
		req *ScheduleFollowUpRequest,
		actor *permissionports.RequestActor,
	) (*AccountView, error)
}
