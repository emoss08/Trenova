package driverpayports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payexpense"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
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

type PayProfileSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
	Classification     driverpay.PayeeClassification
}

type CountActivePayAssignmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ProfileIDs []pulid.ID            `json:"profileIds"`
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
	SelectOptions(
		ctx context.Context,
		req *PayProfileSelectOptionsRequest,
	) (*pagination.ListResult[*driverpay.PayProfile], error)
	GetByID(ctx context.Context, req GetPayProfileByIDRequest) (*driverpay.PayProfile, error)
	Create(ctx context.Context, entity *driverpay.PayProfile) (*driverpay.PayProfile, error)
	Update(ctx context.Context, entity *driverpay.PayProfile) (*driverpay.PayProfile, error)
	CountActiveAssignments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		profileID pulid.ID,
	) (int, error)
	CountActiveAssignmentsByIDs(
		ctx context.Context,
		req CountActivePayAssignmentsRequest,
	) (map[pulid.ID]int, error)
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
	) (*payassignment.WorkerPayAssignment, error)
	GetEffectiveForWorker(
		ctx context.Context,
		req GetWorkerPayAssignmentRequest,
	) (*payassignment.WorkerPayAssignment, error)
	ListForWorker(
		ctx context.Context,
		req ListWorkerPayAssignmentsRequest,
	) ([]*payassignment.WorkerPayAssignment, error)
	ListForProfile(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		profileID pulid.ID,
	) ([]*payassignment.WorkerPayAssignment, error)
	ListOverlapping(
		ctx context.Context,
		entity *payassignment.WorkerPayAssignment,
	) ([]*payassignment.WorkerPayAssignment, error)
	Create(
		ctx context.Context,
		entity *payassignment.WorkerPayAssignment,
	) (*payassignment.WorkerPayAssignment, error)
	Update(
		ctx context.Context,
		entity *payassignment.WorkerPayAssignment,
	) (*payassignment.WorkerPayAssignment, error)
	Delete(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error
}

type GetRecurringDeductionByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListRecurringDeductionsRequest struct {
	Filter   *pagination.QueryOptions     `json:"filter"`
	WorkerID pulid.ID                     `json:"workerId"`
	Status   recurringpay.DeductionStatus `json:"status"`
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
	) (*pagination.ListResult[*recurringpay.RecurringDeduction], error)
	ListConnection(
		ctx context.Context,
		req *ListRecurringDeductionConnectionRequest,
	) (*pagination.CursorListResult[*recurringpay.RecurringDeduction], error)
	GetByID(
		ctx context.Context,
		req GetRecurringDeductionByIDRequest,
	) (*recurringpay.RecurringDeduction, error)
	ListActiveForWorker(
		ctx context.Context,
		req ListActiveDeductionsForWorkerRequest,
	) ([]*recurringpay.RecurringDeduction, error)
	Create(
		ctx context.Context,
		entity *recurringpay.RecurringDeduction,
	) (*recurringpay.RecurringDeduction, error)
	Update(
		ctx context.Context,
		entity *recurringpay.RecurringDeduction,
	) (*recurringpay.RecurringDeduction, error)
}

type GetPayCodeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListPayCodesRequest struct {
	Filter    *pagination.QueryOptions `json:"filter"`
	Direction paycode.PayCodeDirection `json:"direction"`
}

type ListPayCodeConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListActivePayCodesRequest struct {
	TenantInfo pagination.TenantInfo    `json:"tenantInfo"`
	Direction  paycode.PayCodeDirection `json:"direction"`
}

type PayCodeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
	Direction          paycode.PayCodeDirection
}

type PayCodeRepository interface {
	List(
		ctx context.Context,
		req *ListPayCodesRequest,
	) (*pagination.ListResult[*paycode.PayCode], error)
	ListConnection(
		ctx context.Context,
		req *ListPayCodeConnectionRequest,
	) (*pagination.CursorListResult[*paycode.PayCode], error)
	ListActive(
		ctx context.Context,
		req ListActivePayCodesRequest,
	) ([]*paycode.PayCode, error)
	SelectOptions(
		ctx context.Context,
		req *PayCodeSelectOptionsRequest,
	) (*pagination.ListResult[*paycode.PayCode], error)
	GetByID(ctx context.Context, req GetPayCodeByIDRequest) (*paycode.PayCode, error)
	GetByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*paycode.PayCode, error)
	Create(ctx context.Context, entity *paycode.PayCode) (*paycode.PayCode, error)
	Update(ctx context.Context, entity *paycode.PayCode) (*paycode.PayCode, error)
	EnsureSystemDefaults(ctx context.Context, tenantInfo pagination.TenantInfo) error
}

type GetRecurringEarningByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListRecurringEarningsRequest struct {
	Filter   *pagination.QueryOptions   `json:"filter"`
	WorkerID pulid.ID                   `json:"workerId"`
	Status   recurringpay.EarningStatus `json:"status"`
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
	) (*pagination.ListResult[*recurringpay.RecurringEarning], error)
	ListConnection(
		ctx context.Context,
		req *ListRecurringEarningConnectionRequest,
	) (*pagination.CursorListResult[*recurringpay.RecurringEarning], error)
	GetByID(
		ctx context.Context,
		req GetRecurringEarningByIDRequest,
	) (*recurringpay.RecurringEarning, error)
	ListActiveForWorker(
		ctx context.Context,
		req ListActiveEarningsForWorkerRequest,
	) ([]*recurringpay.RecurringEarning, error)
	Create(
		ctx context.Context,
		entity *recurringpay.RecurringEarning,
	) (*recurringpay.RecurringEarning, error)
	Update(
		ctx context.Context,
		entity *recurringpay.RecurringEarning,
	) (*recurringpay.RecurringEarning, error)
}

type GetPayAdvanceByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetPayAdvancesByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	IDs        []pulid.ID            `json:"ids"`
}

type GetEscrowAccountsByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	IDs        []pulid.ID            `json:"ids"`
}

type ListPayAdvancesRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	WorkerID pulid.ID                 `json:"workerId"`
	Status   payadvance.AdvanceStatus `json:"status"`
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
	) (*pagination.ListResult[*payadvance.PayAdvance], error)
	ListConnection(
		ctx context.Context,
		req *ListPayAdvanceConnectionRequest,
	) (*pagination.CursorListResult[*payadvance.PayAdvance], error)
	GetByID(ctx context.Context, req GetPayAdvanceByIDRequest) (*payadvance.PayAdvance, error)
	GetByIDs(
		ctx context.Context,
		req GetPayAdvancesByIDsRequest,
	) ([]*payadvance.PayAdvance, error)
	ListOutstandingForWorker(
		ctx context.Context,
		req ListOutstandingAdvancesForWorkerRequest,
	) ([]*payadvance.PayAdvance, error)
	Create(ctx context.Context, entity *payadvance.PayAdvance) (*payadvance.PayAdvance, error)
	Update(ctx context.Context, entity *payadvance.PayAdvance) (*payadvance.PayAdvance, error)
}

type GetEscrowAccountByIDRequest struct {
	ID                  pulid.ID              `json:"id"`
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	IncludeTransactions bool                  `json:"includeTransactions"`
}

type ListEscrowAccountsRequest struct {
	Filter   *pagination.QueryOptions      `json:"filter"`
	WorkerID pulid.ID                      `json:"workerId"`
	Status   payescrow.EscrowAccountStatus `json:"status"`
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
	) (*pagination.ListResult[*payescrow.EscrowAccount], error)
	ListConnection(
		ctx context.Context,
		req *ListEscrowAccountConnectionRequest,
	) (*pagination.CursorListResult[*payescrow.EscrowAccount], error)
	GetByID(ctx context.Context, req GetEscrowAccountByIDRequest) (*payescrow.EscrowAccount, error)
	GetByIDs(
		ctx context.Context,
		req GetEscrowAccountsByIDsRequest,
	) ([]*payescrow.EscrowAccount, error)
	GetActiveForWorker(
		ctx context.Context,
		req GetActiveEscrowAccountForWorkerRequest,
	) (*payescrow.EscrowAccount, error)
	ListDueForInterest(
		ctx context.Context,
		req ListEscrowAccountsForInterestRequest,
	) ([]*payescrow.EscrowAccount, error)
	Create(ctx context.Context, entity *payescrow.EscrowAccount) (*payescrow.EscrowAccount, error)
	Update(ctx context.Context, entity *payescrow.EscrowAccount) (*payescrow.EscrowAccount, error)
	AppendTransaction(
		ctx context.Context,
		entity *payescrow.EscrowTransaction,
	) (*payescrow.EscrowTransaction, error)
	ListTransactions(
		ctx context.Context,
		req GetEscrowAccountByIDRequest,
	) ([]*payescrow.EscrowTransaction, error)
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
	TenantInfo pagination.TenantInfo      `json:"tenantInfo"`
	WorkerID   pulid.ID                   `json:"workerId"`
	Statuses   []payexpense.ExpenseStatus `json:"statuses"`
	Limit      int                        `json:"limit"`
}

type DriverExpenseRepository interface {
	Create(ctx context.Context, entity *payexpense.Expense) (*payexpense.Expense, error)
	Update(ctx context.Context, entity *payexpense.Expense) (*payexpense.Expense, error)
	GetByID(ctx context.Context, req GetDriverExpenseByIDRequest) (*payexpense.Expense, error)
	ListConnection(
		ctx context.Context,
		req *ListDriverExpenseConnectionRequest,
	) (*pagination.CursorListResult[*payexpense.Expense], error)
	ListForWorker(
		ctx context.Context,
		req *ListDriverExpensesForWorkerRequest,
	) ([]*payexpense.Expense, error)
	ListApprovedForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) ([]*payexpense.Expense, error)
	CountPending(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error)
}
