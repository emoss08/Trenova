package accountingdriftservice

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func (f *fakeRecords) Enqueue(
	_ context.Context,
	records []*accountingsync.AccountingSyncRecord,
) (*repositories.EnqueueAccountingSyncRecordsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := &repositories.EnqueueAccountingSyncRecordsResult{}
	for _, record := range records {
		duplicate := slices.ContainsFunc(f.rows, func(row *accountingsync.AccountingSyncRecord) bool {
			return row.ConnectionID == record.ConnectionID && row.IdempotencyKey == record.IdempotencyKey
		})
		if duplicate {
			result.Existing++
			continue
		}
		stored := *record
		f.rows = append(f.rows, &stored)
		result.Inserted = append(result.Inserted, record)
	}
	return result, nil
}

func (f *fakeRecords) SupersedeOlder(
	_ context.Context,
	req *repositories.SupersedeAccountingSyncRecordsRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, row := range f.rows {
		if row.ObjectID == req.ObjectID && row.Operation == req.Operation &&
			row.Revision < req.BeforeRevision && row.Supersede() {
			count++
		}
	}
	return count, nil
}

func (f *fakeRecords) GetByIDs(
	_ context.Context,
	req repositories.GetAccountingSyncRecordsByIDsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if slices.Contains(req.IDs, row.ID) {
			copied := *row
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeRecords) Update(
	_ context.Context,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for idx, row := range f.rows {
		if row.ID == record.ID {
			copied := *record
			f.rows[idx] = &copied
			return record, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Accounting sync record not found")
}

func (f *fakeRecords) byID(id pulid.ID) *accountingsync.AccountingSyncRecord {
	for _, row := range f.all() {
		if row.ID == id {
			return row
		}
	}
	return nil
}

func (f *fakeRecords) forObject(objectID pulid.ID) []*accountingsync.AccountingSyncRecord {
	out := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.all() {
		if row.ObjectID == objectID {
			out = append(out, row)
		}
	}
	return out
}

type fakePermissions struct {
	services.PermissionEngine
	mu     sync.Mutex
	denied map[string]bool
	checks []string
}

func (f *fakePermissions) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checks = append(f.checks, req.Resource)
	return &services.PermissionCheckResult{Allowed: !f.denied[req.Resource]}, nil
}

type fakeInvoiceRepo struct {
	repositories.InvoiceRepository
	rows map[pulid.ID]*invoice.Invoice
}

func (f *fakeInvoiceRepo) GetByID(
	_ context.Context,
	req repositories.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	inv, ok := f.rows[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Invoice not found")
	}
	copied := *inv
	return &copied, nil
}

type bookEnqueuer struct {
	records *fakeRecords
	conns   []*accountingsync.AccountingConnection
	at      int64
}

func (b *bookEnqueuer) enqueue(
	ctx context.Context,
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
	operation accountingsync.SyncOperation,
	source accountingsync.SyncSourceEvent,
) {
	inserted := make([]*accountingsync.AccountingSyncRecord, 0, len(b.conns))
	for _, conn := range b.conns {
		record := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
			TenantInfo:   pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
			ConnectionID: conn.ID,
			Key: accountingsync.SyncRecordKey{
				ObjectType: objectType,
				ObjectID:   objectID,
				Operation:  operation,
				Revision:   1,
			},
			SourceEvent:  source,
			AwaitRelease: !conn.AutoSync,
			At:           b.at,
		})
		b.records.add(record)
		inserted = append(inserted, record)
	}
	services.NoteAccountingSyncEnqueued(ctx, inserted)
}

type createdMemo struct {
	req   services.CreateMemoRequest
	actor *services.RequestActor
	memo  *invoice.Invoice
}

type fakeInvoices struct {
	services.InvoiceService
	book            *bookEnqueuer
	memos           []createdMemo
	voids           []services.VoidInvoiceRequest
	pendingApproval bool
	failWith        error
}

func (f *fakeInvoices) CreateMemo(
	ctx context.Context,
	req *services.CreateMemoRequest,
	actor *services.RequestActor,
) (*invoice.Invoice, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	memo := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		BillType:           req.BillType,
		CustomerID:         req.CustomerID,
		ReferenceInvoiceID: req.ReferenceInvoiceID,
		Status:             invoice.StatusPosted,
	}
	objectType := accountingsync.SyncObjectDebitMemo
	source := accountingsync.SyncSourceDebitMemoPosted
	if req.BillType == billingqueue.BillTypeCreditMemo {
		objectType = accountingsync.SyncObjectCreditMemo
		source = accountingsync.SyncSourceCreditMemoPosted
	}
	f.book.enqueue(ctx, objectType, memo.ID, accountingsync.SyncOperationCreate, source)
	f.memos = append(f.memos, createdMemo{req: *req, actor: actor, memo: memo})
	return memo, nil
}

func (f *fakeInvoices) VoidInvoice(
	ctx context.Context,
	req *services.VoidInvoiceRequest,
	_ *services.RequestActor,
) (*services.VoidInvoiceResult, error) {
	f.voids = append(f.voids, *req)
	if f.pendingApproval {
		return &services.VoidInvoiceResult{PendingApproval: true}, nil
	}
	f.book.enqueue(ctx, accountingsync.SyncObjectCreditMemo, pulid.MustNew("inv_"),
		accountingsync.SyncOperationCreate, accountingsync.SyncSourceAdjustmentCreditMemo)
	f.book.enqueue(ctx, accountingsync.SyncObjectInvoice, req.InvoiceID,
		accountingsync.SyncOperationVoid, accountingsync.SyncSourceAdjustmentCreditMemo)
	return &services.VoidInvoiceResult{Invoice: &invoice.Invoice{ID: req.InvoiceID}}, nil
}

type fakePayments struct {
	services.CustomerPaymentService
	book     *bookEnqueuer
	rows     map[pulid.ID]*customerpayment.Payment
	reversed []services.ReverseCustomerPaymentRequest
	applied  []services.ApplyCreditMemoRequest
}

func (f *fakePayments) Get(
	_ context.Context,
	req *services.GetCustomerPaymentRequest,
) (*customerpayment.Payment, error) {
	payment, ok := f.rows[req.PaymentID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Payment not found")
	}
	copied := *payment
	return &copied, nil
}

func (f *fakePayments) Reverse(
	ctx context.Context,
	req *services.ReverseCustomerPaymentRequest,
	_ *services.RequestActor,
) (*customerpayment.Payment, error) {
	f.reversed = append(f.reversed, *req)
	f.book.enqueue(ctx, accountingsync.SyncObjectCustomerPayment, req.PaymentID,
		accountingsync.SyncOperationVoid, accountingsync.SyncSourceCustomerPaymentReversed)
	return &customerpayment.Payment{ID: req.PaymentID, Status: customerpayment.StatusReversed}, nil
}

func (f *fakePayments) ApplyCreditMemo(
	ctx context.Context,
	req *services.ApplyCreditMemoRequest,
	_ *services.RequestActor,
) ([]*customerpayment.CreditMemoApplication, error) {
	f.applied = append(f.applied, *req)
	applicationID := pulid.MustNew("cma_")
	f.book.enqueue(ctx, accountingsync.SyncObjectCreditApplication, applicationID,
		accountingsync.SyncOperationCreate, accountingsync.SyncSourceCreditMemoApplied)
	return []*customerpayment.CreditMemoApplication{{ID: applicationID}}, nil
}

type fakeControls struct {
	repositories.AccountingControlRepository
	tolerance string
}

func (f *fakeControls) GetByOrgID(context.Context, pulid.ID) (*tenant.AccountingControl, error) {
	return &tenant.AccountingControl{
		ReconciliationToleranceAmount: decimal.RequireFromString(f.tolerance),
	}, nil
}

type fakeAudit struct {
	services.AuditService
	mu      sync.Mutex
	entries []*services.LogActionParams
}

func (f *fakeAudit) LogAction(params *services.LogActionParams, _ ...services.LogOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, params)
	return nil
}

type fakeDispatcher struct {
	services.AccountingSyncDispatcher
	mu    sync.Mutex
	kicks int
}

func (f *fakeDispatcher) Kick(context.Context, pagination.TenantInfo, pulid.ID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kicks++
	return nil
}

var errBooks = errors.New("the books refused")
