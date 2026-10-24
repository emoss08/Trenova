package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type fakeAuditService struct {
	auditports.AuditService
	logged []*auditports.LogActionParams
}

func (f *fakeAuditService) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	f.logged = append(f.logged, params)
	return nil
}

type fakeEscrowRepo struct {
	driverpayports.EscrowAccountRepository
	getByID func(
		ctx context.Context,
		req driverpayports.GetEscrowAccountByIDRequest,
	) (*driverpay.EscrowAccount, error)
	getActiveForWorker func(
		ctx context.Context,
		req driverpayports.GetActiveEscrowAccountForWorkerRequest,
	) (*driverpay.EscrowAccount, error)
	create func(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
	) (*driverpay.EscrowAccount, error)
	update func(
		ctx context.Context,
		entity *driverpay.EscrowAccount,
	) (*driverpay.EscrowAccount, error)
	appendTransaction func(
		ctx context.Context,
		entity *driverpay.EscrowTransaction,
	) (*driverpay.EscrowTransaction, error)
}

func (f *fakeEscrowRepo) GetByID(
	ctx context.Context,
	req driverpayports.GetEscrowAccountByIDRequest,
) (*driverpay.EscrowAccount, error) {
	return f.getByID(ctx, req)
}

func (f *fakeEscrowRepo) GetActiveForWorker(
	ctx context.Context,
	req driverpayports.GetActiveEscrowAccountForWorkerRequest,
) (*driverpay.EscrowAccount, error) {
	return f.getActiveForWorker(ctx, req)
}

func (f *fakeEscrowRepo) Create(
	ctx context.Context,
	entity *driverpay.EscrowAccount,
) (*driverpay.EscrowAccount, error) {
	return f.create(ctx, entity)
}

func (f *fakeEscrowRepo) Update(
	ctx context.Context,
	entity *driverpay.EscrowAccount,
) (*driverpay.EscrowAccount, error) {
	return f.update(ctx, entity)
}

func (f *fakeEscrowRepo) AppendTransaction(
	ctx context.Context,
	entity *driverpay.EscrowTransaction,
) (*driverpay.EscrowTransaction, error) {
	return f.appendTransaction(ctx, entity)
}

type fakeSettlementControlRepo struct {
	tenantports.SettlementControlRepository
	getOrCreate func(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*tenant.SettlementControl, error)
}

func (f *fakeSettlementControlRepo) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.SettlementControl, error) {
	return f.getOrCreate(ctx, tenantInfo)
}

type fakeAssignmentRepo struct {
	driverpayports.WorkerPayAssignmentRepository
	getByID func(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*driverpay.WorkerPayAssignment, error)
	listOverlapping func(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) ([]*driverpay.WorkerPayAssignment, error)
	create func(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) (*driverpay.WorkerPayAssignment, error)
	update func(
		ctx context.Context,
		entity *driverpay.WorkerPayAssignment,
	) (*driverpay.WorkerPayAssignment, error)
}

func (f *fakeAssignmentRepo) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*driverpay.WorkerPayAssignment, error) {
	return f.getByID(ctx, tenantInfo, id)
}

func (f *fakeAssignmentRepo) ListOverlapping(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) ([]*driverpay.WorkerPayAssignment, error) {
	return f.listOverlapping(ctx, entity)
}

func (f *fakeAssignmentRepo) Create(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) (*driverpay.WorkerPayAssignment, error) {
	return f.create(ctx, entity)
}

func (f *fakeAssignmentRepo) Update(
	ctx context.Context,
	entity *driverpay.WorkerPayAssignment,
) (*driverpay.WorkerPayAssignment, error) {
	return f.update(ctx, entity)
}

type fakeProfileRepo struct {
	driverpayports.PayProfileRepository
	getByID func(
		ctx context.Context,
		req driverpayports.GetPayProfileByIDRequest,
	) (*driverpay.PayProfile, error)
}

func (f *fakeProfileRepo) GetByID(
	ctx context.Context,
	req driverpayports.GetPayProfileByIDRequest,
) (*driverpay.PayProfile, error) {
	return f.getByID(ctx, req)
}

type fakeAdvanceRepo struct {
	driverpayports.PayAdvanceRepository
	getByID func(
		ctx context.Context,
		req driverpayports.GetPayAdvanceByIDRequest,
	) (*driverpay.PayAdvance, error)
	create func(
		ctx context.Context,
		entity *driverpay.PayAdvance,
	) (*driverpay.PayAdvance, error)
	update func(
		ctx context.Context,
		entity *driverpay.PayAdvance,
	) (*driverpay.PayAdvance, error)
}

func (f *fakeAdvanceRepo) GetByID(
	ctx context.Context,
	req driverpayports.GetPayAdvanceByIDRequest,
) (*driverpay.PayAdvance, error) {
	return f.getByID(ctx, req)
}

func (f *fakeAdvanceRepo) Create(
	ctx context.Context,
	entity *driverpay.PayAdvance,
) (*driverpay.PayAdvance, error) {
	return f.create(ctx, entity)
}

func (f *fakeAdvanceRepo) Update(
	ctx context.Context,
	entity *driverpay.PayAdvance,
) (*driverpay.PayAdvance, error) {
	return f.update(ctx, entity)
}

func newTestService() (*Service, *fakeAuditService) {
	audit := &fakeAuditService{}
	svc := &Service{
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		auditService: audit,
	}
	return svc, audit
}

func testActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
}
