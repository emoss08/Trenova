package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetPayProfileByIDRequest struct {
	ID                pulid.ID              `json:"id"`
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
	IncludeComponents bool                  `json:"includeComponents"`
}

type ListPayProfilesRequest struct {
	Filter         *pagination.QueryOptions      `json:"filter"`
	Classification driverpay.PayeeClassification `json:"classification"`
}

type ListPayProfileConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type PayProfileRepository interface {
	List(
		ctx context.Context,
		req *ListPayProfilesRequest,
	) (*pagination.ListResult[*driverpay.PayProfile], error)
	ListConnection(
		ctx context.Context,
		req *ListPayProfileConnectionRequest,
	) (*pagination.CursorListResult[*driverpay.PayProfile], error)
	GetByID(ctx context.Context, req GetPayProfileByIDRequest) (*driverpay.PayProfile, error)
	Create(ctx context.Context, entity *driverpay.PayProfile) (*driverpay.PayProfile, error)
	Update(ctx context.Context, entity *driverpay.PayProfile) (*driverpay.PayProfile, error)
	CountActiveAssignments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		profileID pulid.ID,
	) (int, error)
}

type GetWorkerPayAssignmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	AsOf       int64                 `json:"asOf"`
}

type ListWorkerPayAssignmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type WorkerPayAssignmentRepository interface {
	GetByID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*driverpay.WorkerPayAssignment, error)
	GetEffectiveForWorker(
		ctx context.Context,
		req GetWorkerPayAssignmentRequest,
	) (*driverpay.WorkerPayAssignment, error)
	ListForWorker(
		ctx context.Context,
		req ListWorkerPayAssignmentsRequest,
	) ([]*driverpay.WorkerPayAssignment, error)
	ListForProfile(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		profileID pulid.ID,
	) ([]*driverpay.WorkerPayAssignment, error)
	ListOverlapping(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) ([]*driverpay.WorkerPayAssignment, error)
	Create(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) (*driverpay.WorkerPayAssignment, error)
	Update(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) (*driverpay.WorkerPayAssignment, error)
	Delete(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error
}

type GetRecurringDeductionByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListRecurringDeductionsRequest struct {
	Filter   *pagination.QueryOptions  `json:"filter"`
	WorkerID pulid.ID                  `json:"workerId"`
	Status   driverpay.DeductionStatus `json:"status"`
}

type ListRecurringDeductionConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListActiveDeductionsForWorkerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	AsOf       int64                 `json:"asOf"`
}

type RecurringDeductionRepository interface {
	List(
		ctx context.Context,
		req *ListRecurringDeductionsRequest,
	) (*pagination.ListResult[*driverpay.RecurringDeduction], error)
	ListConnection(
		ctx context.Context,
		req *ListRecurringDeductionConnectionRequest,
	) (*pagination.CursorListResult[*driverpay.RecurringDeduction], error)
	GetByID(
		ctx context.Context,
		req GetRecurringDeductionByIDRequest,
	) (*driverpay.RecurringDeduction, error)
	ListActiveForWorker(
		ctx context.Context,
		req ListActiveDeductionsForWorkerRequest,
	) ([]*driverpay.RecurringDeduction, error)
	Create(
		ctx context.Context,
		entity *driverpay.RecurringDeduction,
	) (*driverpay.RecurringDeduction, error)
	Update(
		ctx context.Context,
		entity *driverpay.RecurringDeduction,
	) (*driverpay.RecurringDeduction, error)
}

type GetPayCodeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListPayCodesRequest struct {
	Filter    *pagination.QueryOptions   `json:"filter"`
	Direction driverpay.PayCodeDirection `json:"direction"`
}

type ListPayCodeConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListActivePayCodesRequest struct {
	TenantInfo pagination.TenantInfo      `json:"tenantInfo"`
	Direction  driverpay.PayCodeDirection `json:"direction"`
}

type PayCodeRepository interface {
	List(
		ctx context.Context,
		req *ListPayCodesRequest,
	) (*pagination.ListResult[*driverpay.PayCode], error)
	ListConnection(
		ctx context.Context,
		req *ListPayCodeConnectionRequest,
	) (*pagination.CursorListResult[*driverpay.PayCode], error)
	ListActive(
		ctx context.Context,
		req ListActivePayCodesRequest,
	) ([]*driverpay.PayCode, error)
	GetByID(ctx context.Context, req GetPayCodeByIDRequest) (*driverpay.PayCode, error)
	GetByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*driverpay.PayCode, error)
	Create(ctx context.Context, entity *driverpay.PayCode) (*driverpay.PayCode, error)
	Update(ctx context.Context, entity *driverpay.PayCode) (*driverpay.PayCode, error)
	EnsureSystemDefaults(ctx context.Context, tenantInfo pagination.TenantInfo) error
}

type GetRecurringEarningByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListRecurringEarningsRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	WorkerID pulid.ID                 `json:"workerId"`
	Status   driverpay.EarningStatus  `json:"status"`
}

type ListRecurringEarningConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListActiveEarningsForWorkerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	AsOf       int64                 `json:"asOf"`
}

type RecurringEarningRepository interface {
	List(
		ctx context.Context,
		req *ListRecurringEarningsRequest,
	) (*pagination.ListResult[*driverpay.RecurringEarning], error)
	ListConnection(
		ctx context.Context,
		req *ListRecurringEarningConnectionRequest,
	) (*pagination.CursorListResult[*driverpay.RecurringEarning], error)
	GetByID(
		ctx context.Context,
		req GetRecurringEarningByIDRequest,
	) (*driverpay.RecurringEarning, error)
	ListActiveForWorker(
		ctx context.Context,
		req ListActiveEarningsForWorkerRequest,
	) ([]*driverpay.RecurringEarning, error)
	Create(
		ctx context.Context,
		entity *driverpay.RecurringEarning,
	) (*driverpay.RecurringEarning, error)
	Update(
		ctx context.Context,
		entity *driverpay.RecurringEarning,
	) (*driverpay.RecurringEarning, error)
}

type GetPayAdvanceByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListPayAdvancesRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	WorkerID pulid.ID                 `json:"workerId"`
	Status   driverpay.AdvanceStatus  `json:"status"`
}

type ListPayAdvanceConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListOutstandingAdvancesForWorkerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type PayAdvanceRepository interface {
	List(
		ctx context.Context,
		req *ListPayAdvancesRequest,
	) (*pagination.ListResult[*driverpay.PayAdvance], error)
	ListConnection(
		ctx context.Context,
		req *ListPayAdvanceConnectionRequest,
	) (*pagination.CursorListResult[*driverpay.PayAdvance], error)
	GetByID(ctx context.Context, req GetPayAdvanceByIDRequest) (*driverpay.PayAdvance, error)
	ListOutstandingForWorker(
		ctx context.Context,
		req ListOutstandingAdvancesForWorkerRequest,
	) ([]*driverpay.PayAdvance, error)
	Create(ctx context.Context, entity *driverpay.PayAdvance) (*driverpay.PayAdvance, error)
	Update(ctx context.Context, entity *driverpay.PayAdvance) (*driverpay.PayAdvance, error)
}

type GetEscrowAccountByIDRequest struct {
	ID                  pulid.ID              `json:"id"`
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	IncludeTransactions bool                  `json:"includeTransactions"`
}

type ListEscrowAccountsRequest struct {
	Filter   *pagination.QueryOptions      `json:"filter"`
	WorkerID pulid.ID                      `json:"workerId"`
	Status   driverpay.EscrowAccountStatus `json:"status"`
}

type ListEscrowAccountConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type GetActiveEscrowAccountForWorkerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type ListEscrowAccountsForInterestRequest struct {
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	AccrueOnOrBefore int64                 `json:"accrueOnOrBefore"`
}

type EscrowAccountRepository interface {
	List(
		ctx context.Context,
		req *ListEscrowAccountsRequest,
	) (*pagination.ListResult[*driverpay.EscrowAccount], error)
	ListConnection(
		ctx context.Context,
		req *ListEscrowAccountConnectionRequest,
	) (*pagination.CursorListResult[*driverpay.EscrowAccount], error)
	GetByID(ctx context.Context, req GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error)
	GetActiveForWorker(
		ctx context.Context,
		req GetActiveEscrowAccountForWorkerRequest,
	) (*driverpay.EscrowAccount, error)
	ListDueForInterest(
		ctx context.Context,
		req ListEscrowAccountsForInterestRequest,
	) ([]*driverpay.EscrowAccount, error)
	Create(ctx context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error)
	Update(ctx context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error)
	AppendTransaction(
		ctx context.Context,
		entity *driverpay.EscrowTransaction,
	) (*driverpay.EscrowTransaction, error)
	ListTransactions(
		ctx context.Context,
		req GetEscrowAccountByIDRequest,
	) ([]*driverpay.EscrowTransaction, error)
}

type GetDriverExpenseByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListDriverExpenseConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListDriverExpensesForWorkerRequest struct {
	TenantInfo pagination.TenantInfo     `json:"tenantInfo"`
	WorkerID   pulid.ID                  `json:"workerId"`
	Statuses   []driverpay.ExpenseStatus `json:"statuses"`
	Limit      int                       `json:"limit"`
}

type DriverExpenseRepository interface {
	Create(ctx context.Context, entity *driverpay.Expense) (*driverpay.Expense, error)
	Update(ctx context.Context, entity *driverpay.Expense) (*driverpay.Expense, error)
	GetByID(ctx context.Context, req GetDriverExpenseByIDRequest) (*driverpay.Expense, error)
	ListConnection(
		ctx context.Context,
		req *ListDriverExpenseConnectionRequest,
	) (*pagination.CursorListResult[*driverpay.Expense], error)
	ListForWorker(
		ctx context.Context,
		req *ListDriverExpensesForWorkerRequest,
	) ([]*driverpay.Expense, error)
	ListApprovedForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) ([]*driverpay.Expense, error)
	CountPending(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error)
}
