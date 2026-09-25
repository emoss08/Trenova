package accountinginboundservice

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type fakeConnections struct {
	repositories.AccountingConnectionRepository
	mu    sync.Mutex
	conn  *accountingsync.AccountingConnection
	saves []repositories.SaveAccountingChangeFeedRequest
}

func (f *fakeConnections) current() *accountingsync.AccountingConnection {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := *f.conn
	return &out
}

func (f *fakeConnections) GetByID(
	_ context.Context,
	req repositories.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	if req.ID != f.conn.ID {
		return nil, errortypes.NewNotFoundError("Accounting connection not found")
	}
	return f.current(), nil
}

func (f *fakeConnections) GetByType(
	_ context.Context,
	_ repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	return f.current(), nil
}

func (f *fakeConnections) SaveChangeFeed(
	_ context.Context,
	req *repositories.SaveAccountingChangeFeedRequest,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saves = append(f.saves, *req)
	if req.OnlyIfEmpty && f.conn.ChangeCursor != "" {
		return false, nil
	}
	if req.Cursor != "" {
		f.conn.ChangeCursor = req.Cursor
	}
	if req.ReadAt != nil {
		at := *req.ReadAt
		f.conn.ChangesReadAt = &at
	}
	f.conn.ChangesErrorCategory = req.ErrorCategory
	f.conn.ChangesErrorMessage = req.ErrorMessage
	return true, nil
}

type fakeConnector struct {
	services.AccountingConnector
	mu      sync.Mutex
	pages   []*services.AccountingChangePage
	readErr error
	reads   []services.ReadAccountingChangesRequest
}

func (f *fakeConnector) ChangeFeedLimits() services.AccountingChangeFeedLimits {
	return services.AccountingChangeFeedLimits{MaxLookback: 30 * 24 * time.Hour}
}

func (f *fakeConnector) ChangeCursorAt(at time.Time) string {
	return "at:" + at.UTC().Format(time.RFC3339)
}

func (f *fakeConnector) ReadChanges(
	_ context.Context,
	req *services.ReadAccountingChangesRequest,
) (*services.AccountingChangePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads = append(f.reads, *req)
	if f.readErr != nil {
		return nil, f.readErr
	}
	if len(f.pages) == 0 {
		return &services.AccountingChangePage{NextCursor: req.Cursor}, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

func (f *fakeConnector) DocumentURL(kind accountingsync.SyncObjectType, externalID string) string {
	return "https://books.example/" + string(kind) + "/" + externalID
}

func (f *fakeConnector) ClassifyDocumentError(err error) *accountingsync.SyncError {
	return &accountingsync.SyncError{Category: accountingsync.SyncErrorAuth, Message: err.Error()}
}

type fakeConnService struct {
	services.AccountingConnectionService
	connections *fakeConnections
	connector   *fakeConnector
	failSession bool
}

func (f *fakeConnService) Session(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*services.AccountingSession, error) {
	if f.failSession {
		return nil, errors.New("the connection needs to be reconnected")
	}
	return &services.AccountingSession{
		Connection:  f.connections.current(),
		AccessToken: "token",
		Connector:   writerConnector{fakeConnector: f.connector},
	}, nil
}

type writerConnector struct {
	*fakeConnector
	services.AccountingDocumentWriter
}

func (w writerConnector) DocumentURL(kind accountingsync.SyncObjectType, externalID string) string {
	return w.fakeConnector.DocumentURL(kind, externalID)
}

func (w writerConnector) ClassifyDocumentError(err error) *accountingsync.SyncError {
	return w.fakeConnector.ClassifyDocumentError(err)
}

type fakeRecords struct {
	repositories.AccountingSyncRecordRepository
	mu   sync.Mutex
	rows []*accountingsync.AccountingSyncRecord
}

func (f *fakeRecords) add(record *accountingsync.AccountingSyncRecord) *accountingsync.AccountingSyncRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, record)
	return record
}

func (f *fakeRecords) all() []*accountingsync.AccountingSyncRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*accountingsync.AccountingSyncRecord{}, f.rows...)
}

func (f *fakeRecords) ListByExternalIDs(
	_ context.Context,
	req *repositories.ListAccountingSyncRecordsByExternalIDsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if row.ConnectionID != req.ConnectionID || row.ExternalID == "" ||
			!slices.Contains(req.ExternalIDs, row.ExternalID) {
			continue
		}
		if len(req.ObjectTypes) > 0 && !slices.Contains(req.ObjectTypes, row.ObjectType) {
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeRecords) ListByObjects(
	_ context.Context,
	req *repositories.ListAccountingSyncRecordsByObjectsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if row.ConnectionID != req.ConnectionID || !slices.Contains(req.ObjectIDs, row.ObjectID) {
			continue
		}
		if len(req.ObjectTypes) > 0 && !slices.Contains(req.ObjectTypes, row.ObjectType) {
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeRecords) CountInFlight(
	_ context.Context,
	req *repositories.CountAccountingSyncInFlightRequest,
) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID && row.Status == accountingsync.SyncStatusInFlight &&
			slices.Contains(req.ObjectTypes, row.ObjectType) {
			count++
		}
	}
	return count, nil
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

type fakeChanges struct {
	repositories.AccountingInboundChangeRepository
	mu   sync.Mutex
	rows []*accountingsync.AccountingInboundChange
}

func cloneChange(c *accountingsync.AccountingInboundChange) *accountingsync.AccountingInboundChange {
	out := *c
	out.Document.Lines = make([]*accountingsync.InboundLine, 0, len(c.Document.Lines))
	for _, line := range c.Document.Lines {
		copied := *line
		out.Document.Lines = append(out.Document.Lines, &copied)
	}
	out.AppliedObjects = append([]accountingsync.AppliedObject{}, c.AppliedObjects...)
	return &out
}

func (f *fakeChanges) all() []*accountingsync.AccountingInboundChange {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingInboundChange, 0, len(f.rows))
	for _, row := range f.rows {
		out = append(out, cloneChange(row))
	}
	return out
}

func (f *fakeChanges) byExternal(externalID string) *accountingsync.AccountingInboundChange {
	for _, row := range f.all() {
		if row.ExternalID == externalID {
			return row
		}
	}
	return nil
}

func (f *fakeChanges) put(change *accountingsync.AccountingInboundChange) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, cloneChange(change))
}

func (f *fakeChanges) Create(
	_ context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	f.put(entity)
	return entity, nil
}

func (f *fakeChanges) Update(
	_ context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for idx, row := range f.rows {
		if row.ID == entity.ID {
			if row.Version != entity.Version {
				return nil, errortypes.NewConflictError("version mismatch")
			}
			entity.Version++
			f.rows[idx] = cloneChange(entity)
			return entity, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Accounting inbound change not found")
}

func (f *fakeChanges) GetByID(
	_ context.Context,
	req repositories.GetAccountingInboundChangeRequest,
) (*accountingsync.AccountingInboundChange, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.ID == req.ID && row.OrganizationID == req.TenantInfo.OrgID &&
			row.BusinessUnitID == req.TenantInfo.BuID {
			return cloneChange(row), nil
		}
	}
	return nil, errortypes.NewNotFoundError("Accounting inbound change not found")
}

func (f *fakeChanges) ListByExternalIDs(
	_ context.Context,
	req *repositories.ListAccountingInboundChangesByExternalIDsRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	out := []*accountingsync.AccountingInboundChange{}
	for _, row := range f.all() {
		if row.ConnectionID == req.ConnectionID && row.Kind == req.Kind &&
			slices.Contains(req.ExternalIDs, row.ExternalID) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeChanges) ListDetected(
	_ context.Context,
	req *repositories.ListDetectedAccountingInboundChangesRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	out := []*accountingsync.AccountingInboundChange{}
	for _, row := range f.all() {
		if row.ConnectionID == req.ConnectionID && row.Status == accountingsync.InboundStatusDetected &&
			row.DetectedAt <= req.DetectedBefore {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeChanges) ListAttention(
	_ context.Context,
	req *repositories.ListAccountingInboundAttentionRequest,
) ([]repositories.AccountingInboundAttentionGroup, error) {
	byReason := map[accountingsync.InboundChangeReason]*repositories.AccountingInboundAttentionGroup{}
	order := []accountingsync.InboundChangeReason{}
	for _, row := range f.all() {
		if row.ConnectionID != req.ConnectionID || row.Status != accountingsync.InboundStatusProposed ||
			row.DetectedAt > req.DetectedBefore {
			continue
		}
		group := byReason[row.Reason]
		if group == nil {
			group = &repositories.AccountingInboundAttentionGroup{
				Reason:           row.Reason,
				OldestDetectedAt: row.DetectedAt,
				SampleID:         row.ID,
			}
			byReason[row.Reason] = group
			order = append(order, row.Reason)
		}
		group.Count++
	}
	out := make([]repositories.AccountingInboundAttentionGroup, 0, len(order))
	for _, reason := range order {
		out = append(out, *byReason[reason])
	}
	return out, nil
}

type fakeReferences struct {
	repositories.AccountingReferenceObjectRepository
	mu       sync.Mutex
	upserted []*accountingsync.AccountingReferenceObject
	removed  map[accountingsync.ReferenceKind][]string
}

func (f *fakeReferences) Upsert(
	_ context.Context,
	req *repositories.UpsertAccountingReferenceObjectsRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.upserted = append(f.upserted, req.Objects...)
	return nil
}

func (f *fakeReferences) MarkRemoved(
	_ context.Context,
	req *repositories.MarkAccountingReferencesRemovedRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.removed == nil {
		f.removed = map[accountingsync.ReferenceKind][]string{}
	}
	f.removed[req.Kind] = append(f.removed[req.Kind], req.ExternalIDs...)
	return int64(len(req.ExternalIDs)), nil
}

type fakeMappings struct {
	repositories.AccountingMappingRepository
	rows []*accountingsync.AccountingMapping
}

func (f *fakeMappings) ListByConnection(
	_ context.Context,
	req *repositories.ListAccountingMappingsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	out := []*accountingsync.AccountingMapping{}
	for _, row := range f.rows {
		if slices.Contains(req.TargetTypes, row.TargetType) {
			out = append(out, row)
		}
	}
	return out, nil
}

type fakeInvoices struct {
	repositories.InvoiceRepository
	mu   sync.Mutex
	rows map[pulid.ID]*invoice.Invoice
}

func (f *fakeInvoices) GetByIDs(
	_ context.Context,
	req repositories.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*invoice.Invoice{}
	for _, id := range req.InvoiceIDs {
		if row, ok := f.rows[id]; ok {
			copied := *row
			out = append(out, &copied)
		}
	}
	return out, nil
}

type fakePayables struct {
	rows map[pulid.ID]*repositories.PayableSettlement
}

func (f *fakePayables) GetSettlement(
	_ context.Context,
	req *repositories.GetPayableSettlementRequest,
) (*repositories.PayableSettlement, error) {
	row, ok := f.rows[req.ID]
	if !ok || row.Kind != req.Kind {
		return nil, errortypes.NewNotFoundError("Settlement not found")
	}
	copied := *row
	return &copied, nil
}

type fakePeriods struct {
	repositories.FiscalPeriodRepository
	status fiscalperiod.Status
	none   bool
	asked  []int64
}

func (f *fakePeriods) GetPeriodByDate(
	_ context.Context,
	req repositories.GetPeriodByDateRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	f.asked = append(f.asked, req.Date)
	if f.none {
		return nil, errortypes.NewNotFoundError("Fiscal period not found")
	}
	return &fiscalperiod.FiscalPeriod{ID: pulid.MustNew("fp_"), Status: f.status}, nil
}

type fakeOrganizations struct {
	repositories.OrganizationRepository
	timezone string
}

func (f *fakeOrganizations) GetByID(
	_ context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{ID: req.TenantInfo.OrgID, Timezone: f.timezone}, nil
}

type fakeUsers struct {
	repositories.UserRepository
	system *tenant.User
}

func (f *fakeUsers) GetSystemUser(context.Context, ...string) (*tenant.User, error) {
	return f.system, nil
}

type postedPayment struct {
	req   services.PostCustomerPaymentRequest
	actor services.RequestActor
}

type appliedCredit struct {
	req   services.ApplyCreditMemoRequest
	actor services.RequestActor
}

type fakeCustomerPayments struct {
	services.CustomerPaymentService
	mu        sync.Mutex
	records   *fakeRecords
	conn      *accountingsync.AccountingConnection
	posted    []postedPayment
	credits   []appliedCredit
	postErr   error
	enqueueAt int64
}

func (f *fakeCustomerPayments) queue(objectType accountingsync.SyncObjectType, id pulid.ID) {
	f.records.add(accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   pagination.TenantInfo{OrgID: f.conn.OrganizationID, BuID: f.conn.BusinessUnitID},
		ConnectionID: f.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: objectType,
			ObjectID:   id,
			Operation:  accountingsync.SyncOperationCreate,
		},
		SourceEvent: accountingsync.SyncSourceCustomerPaymentPosted,
		At:          f.enqueueAt,
	}))
}

func (f *fakeCustomerPayments) PostAndApply(
	_ context.Context,
	req *services.PostCustomerPaymentRequest,
	actor *services.RequestActor,
) (*customerpayment.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.postErr != nil {
		return nil, f.postErr
	}
	f.posted = append(f.posted, postedPayment{req: *req, actor: *actor})
	payment := &customerpayment.Payment{ID: pulid.MustNew("cpay_"), CustomerID: req.CustomerID}
	f.queue(accountingsync.SyncObjectCustomerPayment, payment.ID)
	return payment, nil
}

func (f *fakeCustomerPayments) ApplyCreditMemo(
	_ context.Context,
	req *services.ApplyCreditMemoRequest,
	actor *services.RequestActor,
) ([]*customerpayment.CreditMemoApplication, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.credits = append(f.credits, appliedCredit{req: *req, actor: *actor})
	out := make([]*customerpayment.CreditMemoApplication, 0, len(req.Applications))
	for range req.Applications {
		application := &customerpayment.CreditMemoApplication{ID: pulid.MustNew("cmapp_")}
		f.queue(accountingsync.SyncObjectCreditApplication, application.ID)
		out = append(out, application)
	}
	return out, nil
}

type paidSettlement struct {
	req   services.MarkSettlementPaidRequest
	actor services.RequestActor
}

type fakeCarrierPayer struct {
	mu      sync.Mutex
	records *fakeRecords
	conn    *accountingsync.AccountingConnection
	paid    []paidSettlement
}

func (f *fakeCarrierPayer) MarkPaid(
	_ context.Context,
	req *services.MarkSettlementPaidRequest,
	actor *services.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.paid = append(f.paid, paidSettlement{req: *req, actor: *actor})
	f.records.add(accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   req.TenantInfo,
		ConnectionID: f.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectCarrierBillPay,
			ObjectID:   req.SettlementID,
			Operation:  accountingsync.SyncOperationCreate,
		},
		SourceEvent: accountingsync.SyncSourceCarrierSettlementPaid,
		At:          req.PaidAt,
	}))
	return &carriersettlement.CarrierSettlement{ID: req.SettlementID}, nil
}

type fakeDriverPayer struct {
	paid []paidSettlement
}

func (f *fakeDriverPayer) MarkPaid(
	_ context.Context,
	req *services.MarkSettlementPaidRequest,
	actor *services.RequestActor,
) (*driversettlement.Settlement, error) {
	f.paid = append(f.paid, paidSettlement{req: *req, actor: *actor})
	return &driversettlement.Settlement{ID: req.SettlementID}, nil
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

type fakeWatchtower struct {
	mu       sync.Mutex
	open     map[string]services.WatchtowerItemInput
	resolved []string
}

func (f *fakeWatchtower) Upsert(_ context.Context, input services.WatchtowerItemInput) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.open == nil {
		f.open = map[string]services.WatchtowerItemInput{}
	}
	f.open[input.SourceID] = input
}

func (f *fakeWatchtower) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	_ watchtower.SourceKind,
	sourceID string,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.open, sourceID)
	f.resolved = append(f.resolved, sourceID)
}

type fakeRefresher struct {
	requested int
}

func (f *fakeRefresher) RequestReferenceRefresh(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) error {
	f.requested++
	return nil
}
