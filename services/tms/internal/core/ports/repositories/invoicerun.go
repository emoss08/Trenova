package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetInvoiceRunByIDRequest struct {
	ID            pulid.ID              `json:"-"`
	TenantInfo    pagination.TenantInfo `json:"-"`
	IncludeGroups bool                  `json:"-"`
	IncludeItems  bool                  `json:"-"`
}

type ListInvoiceRunsRequest struct {
	Filter *pagination.QueryOptions `json:"-"`
}

type ListInvoiceRunConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"-"`
}

// GetOpenScheduledRunRequest finds the run a cron tick already produced for a
// period, so a re-fired tick resumes rather than building a second one.
type GetOpenScheduledRunRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Cycle      string                `json:"-"`
	PeriodEnd  int64                 `json:"-"`
}

// ReplaceGroupsRequest swaps a run's whole proposal in one transaction. Building
// is not incremental: a rebuild replaces what was there rather than merging into
// it, so a stale group can never survive a re-preview.
type ReplaceGroupsRequest struct {
	TenantInfo pagination.TenantInfo         `json:"-"`
	RunID      pulid.ID                      `json:"-"`
	Groups     []*invoicerun.InvoiceRunGroup `json:"-"`
}

type InvoiceRunRepository interface {
	List(
		ctx context.Context,
		req *ListInvoiceRunsRequest,
	) (*pagination.ListResult[*invoicerun.InvoiceRun], error)
	ListConnection(
		ctx context.Context,
		req *ListInvoiceRunConnectionRequest,
	) (*pagination.CursorListResult[*invoicerun.InvoiceRun], error)
	GetByID(
		ctx context.Context,
		req GetInvoiceRunByIDRequest,
	) (*invoicerun.InvoiceRun, error)
	GetOpenScheduledRun(
		ctx context.Context,
		req GetOpenScheduledRunRequest,
	) (*invoicerun.InvoiceRun, error)
	Create(
		ctx context.Context,
		entity *invoicerun.InvoiceRun,
	) (*invoicerun.InvoiceRun, error)
	Update(
		ctx context.Context,
		entity *invoicerun.InvoiceRun,
	) (*invoicerun.InvoiceRun, error)
	ReplaceGroups(ctx context.Context, req *ReplaceGroupsRequest) error
	UpdateGroup(
		ctx context.Context,
		entity *invoicerun.InvoiceRunGroup,
	) (*invoicerun.InvoiceRunGroup, error)
	UpdateItems(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		items []*invoicerun.InvoiceRunGroupItem,
	) error
}
