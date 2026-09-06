package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var ErrDuplicatePTOLedgerEntry = errors.New("duplicate PTO ledger entry")

type PTOBalanceKey struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	PTOType    worker.PTOType        `json:"ptoType"`
}

type ListPTOBalancesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type ListPTOLedgerRequest struct {
	Filter        *pagination.QueryOptions `json:"filter"`
	Cursor        pagination.CursorInfo    `json:"cursor"`
	WorkerID      pulid.ID                 `json:"workerId"`
	PTOType       string                   `json:"ptoType"`
	EntryType     string                   `json:"entryType"`
	EffectiveFrom int64                    `json:"effectiveFrom"`
	EffectiveTo   int64                    `json:"effectiveTo"`
	IncludeWorker bool                     `json:"includeWorker"`
}

type PendingPTODaysRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	PTOType    worker.PTOType        `json:"ptoType"`
	ExcludeID  pulid.ID              `json:"excludeId"`
}

type HasPTOLedgerEntryRequest struct {
	TenantInfo  pagination.TenantInfo     `json:"tenantInfo"`
	SourcePTOID pulid.ID                  `json:"sourcePtoId"`
	EntryType   worker.PTOLedgerEntryType `json:"entryType"`
}

type PTOBalanceSummary struct {
	WorkersTracked    int             `json:"workersTracked"`
	WorkersUnassigned int             `json:"workersUnassigned"`
	TotalBalanceDays  decimal.Decimal `json:"totalBalanceDays"`
	TotalPendingDays  decimal.Decimal `json:"totalPendingDays"`
	AccruedYTDDays    decimal.Decimal `json:"accruedYtdDays"`
	UsedYTDDays       decimal.Decimal `json:"usedYtdDays"`
}

type PTOLedgerRepository interface {
	EnsureBalance(ctx context.Context, key *PTOBalanceKey) error
	LockBalance(ctx context.Context, key *PTOBalanceKey) (*worker.WorkerPTOBalance, error)
	GetBalance(ctx context.Context, key *PTOBalanceKey) (*worker.WorkerPTOBalance, error)
	ListBalances(
		ctx context.Context,
		req *ListPTOBalancesRequest,
	) ([]*worker.WorkerPTOBalance, error)
	ListOrgBalances(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*worker.WorkerPTOBalance, error)
	UpdateBalance(
		ctx context.Context,
		entity *worker.WorkerPTOBalance,
	) (*worker.WorkerPTOBalance, error)
	InsertEntry(
		ctx context.Context,
		entry *worker.WorkerPTOLedgerEntry,
	) (*worker.WorkerPTOLedgerEntry, error)
	ListEntries(
		ctx context.Context,
		req *ListPTOLedgerRequest,
	) (*pagination.CursorListResult[*worker.WorkerPTOLedgerEntry], error)
	SumEntries(ctx context.Context, key *PTOBalanceKey) (decimal.Decimal, int64, error)
	PendingDays(ctx context.Context, req *PendingPTODaysRequest) (decimal.Decimal, error)
	HasEntry(ctx context.Context, req *HasPTOLedgerEntryRequest) (bool, error)
	Summary(ctx context.Context, tenantInfo pagination.TenantInfo) (*PTOBalanceSummary, error)
}
