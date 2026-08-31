package driverpayservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fakeDB struct{}

func (fakeDB) DB() *bun.DB { return nil }

func (fakeDB) DBForContext(context.Context) bun.IDB { return nil }

func (fakeDB) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

func (fakeDB) HealthCheck(context.Context) error { return nil }

func (fakeDB) IsHealthy(context.Context) bool { return true }

func (fakeDB) Close() error { return nil }

type fakeAuditService struct {
	serviceports.AuditService
	logged []*serviceports.LogActionParams
}

func (f *fakeAuditService) LogAction(
	params *serviceports.LogActionParams,
	_ ...serviceports.LogOption,
) error {
	f.logged = append(f.logged, params)
	return nil
}

type fakeEscrowRepo struct {
	repositories.EscrowAccountRepository
	getByID func(
		ctx context.Context,
		req repositories.GetEscrowAccountByIDRequest,
	) (*driverpay.EscrowAccount, error)
	getActiveForWorker func(
		ctx context.Context,
		req repositories.GetActiveEscrowAccountForWorkerRequest,
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
	req repositories.GetEscrowAccountByIDRequest,
) (*driverpay.EscrowAccount, error) {
	return f.getByID(ctx, req)
}

func (f *fakeEscrowRepo) GetActiveForWorker(
	ctx context.Context,
	req repositories.GetActiveEscrowAccountForWorkerRequest,
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
	repositories.SettlementControlRepository
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
	repositories.WorkerPayAssignmentRepository
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
	repositories.PayProfileRepository
	getByID func(
		ctx context.Context,
		req repositories.GetPayProfileByIDRequest,
	) (*driverpay.PayProfile, error)
}

func (f *fakeProfileRepo) GetByID(
	ctx context.Context,
	req repositories.GetPayProfileByIDRequest,
) (*driverpay.PayProfile, error) {
	return f.getByID(ctx, req)
}

type fakeAdvanceRepo struct {
	repositories.PayAdvanceRepository
	getByID func(
		ctx context.Context,
		req repositories.GetPayAdvanceByIDRequest,
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
	req repositories.GetPayAdvanceByIDRequest,
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
		db:           fakeDB{},
		auditService: audit,
	}
	return svc, audit
}

func testActor() *serviceports.RequestActor {
	return &serviceports.RequestActor{UserID: pulid.MustNew("usr_")}
}
