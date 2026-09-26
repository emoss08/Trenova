package accountingdriftservice

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type fakeConnections struct {
	repositories.AccountingConnectionRepository
	mu     sync.Mutex
	conn   *accountingsync.AccountingConnection
	checks []repositories.SaveAccountingDriftCheckRequest
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

func (f *fakeConnections) SaveDriftCheck(
	_ context.Context,
	req *repositories.SaveAccountingDriftCheckRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checks = append(f.checks, *req)
	if req.CheckedAt != nil {
		f.conn.DriftCheckedAt = req.CheckedAt
	}
	f.conn.DriftErrorCategory = req.ErrorCategory
	f.conn.DriftErrorMessage = req.ErrorMessage
	return nil
}

type readCall struct {
	kind accountingsync.SyncObjectType
	ids  []string
	refs []map[string]string
}

type fakeReader struct {
	services.AccountingConnector
	mu       sync.Mutex
	maxRead  int
	docs     map[string]*services.AccountingDocumentState
	omit     map[string]bool
	calls    []readCall
	failWith error
}

func (f *fakeReader) set(kind accountingsync.SyncObjectType, state *services.AccountingDocumentState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[string(kind)+":"+state.ExternalID] = state
}

func (f *fakeReader) DocumentReadLimits() services.AccountingDocumentReadLimits {
	return services.AccountingDocumentReadLimits{MaxPerRead: f.maxRead}
}

func (f *fakeReader) ReadDocuments(
	_ context.Context,
	req *services.ReadAccountingDocumentsRequest,
) ([]*services.AccountingDocumentState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return nil, f.failWith
	}
	call := readCall{kind: req.Kind}
	out := make([]*services.AccountingDocumentState, 0, len(req.Targets))
	for _, target := range req.Targets {
		call.ids = append(call.ids, target.ExternalID)
		call.refs = append(call.refs, target.Refs)
		if f.omit[target.ExternalID] {
			continue
		}
		if state, ok := f.docs[string(req.Kind)+":"+target.ExternalID]; ok {
			copied := *state
			out = append(out, &copied)
			continue
		}
		out = append(out, &services.AccountingDocumentState{ExternalID: target.ExternalID})
	}
	f.calls = append(f.calls, call)
	return out, nil
}

func (f *fakeReader) DocumentURL(kind accountingsync.SyncObjectType, externalID string) string {
	return "https://books.example/" + string(kind) + "/" + externalID
}

func (f *fakeReader) ClassifyDocumentError(err error) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorAuth,
		Message:  err.Error(),
	}
}

type readerConnector struct {
	*fakeReader
	services.AccountingDocumentWriter
}

func (r readerConnector) DocumentURL(kind accountingsync.SyncObjectType, externalID string) string {
	return r.fakeReader.DocumentURL(kind, externalID)
}

func (r readerConnector) ClassifyDocumentError(err error) *accountingsync.SyncError {
	return r.fakeReader.ClassifyDocumentError(err)
}

type fakeConnService struct {
	services.AccountingConnectionService
	connections *fakeConnections
	reader      *fakeReader
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
		Connector:   readerConnector{fakeReader: f.reader},
	}, nil
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
			!slices.Contains(req.ExternalIDs, row.ExternalID) ||
			!slices.Contains(req.ObjectTypes, row.ObjectType) {
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
		if row.ConnectionID != req.ConnectionID || !slices.Contains(req.ObjectIDs, row.ObjectID) ||
			!slices.Contains(req.ObjectTypes, row.ObjectType) {
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out, nil
}

type fakeFindings struct {
	repositories.AccountingDriftFindingRepository
	mu   sync.Mutex
	rows []*accountingsync.AccountingDriftFinding
}

func cloneFinding(f *accountingsync.AccountingDriftFinding) *accountingsync.AccountingDriftFinding {
	out := *f
	out.Detail = append([]accountingsync.DriftLine{}, f.Detail...)
	return &out
}

func (f *fakeFindings) all() []*accountingsync.AccountingDriftFinding {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingDriftFinding, 0, len(f.rows))
	for _, row := range f.rows {
		out = append(out, cloneFinding(row))
	}
	return out
}

func (f *fakeFindings) open() []*accountingsync.AccountingDriftFinding {
	out := []*accountingsync.AccountingDriftFinding{}
	for _, row := range f.all() {
		if row.IsOpen() {
			out = append(out, row)
		}
	}
	return out
}

func (f *fakeFindings) Create(
	_ context.Context,
	entity *accountingsync.AccountingDriftFinding,
) (*accountingsync.AccountingDriftFinding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.IsOpen() && row.ConnectionID == entity.ConnectionID &&
			row.ObjectType == entity.ObjectType && row.ObjectID == entity.ObjectID &&
			row.Kind == entity.Kind {
			return nil, errors.New("duplicate open finding")
		}
	}
	f.rows = append(f.rows, cloneFinding(entity))
	return entity, nil
}

func (f *fakeFindings) Update(
	_ context.Context,
	entity *accountingsync.AccountingDriftFinding,
) (*accountingsync.AccountingDriftFinding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for idx, row := range f.rows {
		if row.ID == entity.ID {
			entity.Version++
			f.rows[idx] = cloneFinding(entity)
			return entity, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Drift finding not found")
}

func (f *fakeFindings) GetByID(
	_ context.Context,
	req repositories.GetAccountingDriftFindingRequest,
) (*accountingsync.AccountingDriftFinding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.ID == req.ID && row.OrganizationID == req.TenantInfo.OrgID {
			return cloneFinding(row), nil
		}
	}
	return nil, errortypes.NewNotFoundError("Drift finding not found")
}

func (f *fakeFindings) ListOpen(
	_ context.Context,
	req *repositories.ListOpenAccountingDriftFindingsRequest,
) ([]*accountingsync.AccountingDriftFinding, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingDriftFinding{}
	for _, row := range f.rows {
		if row.IsOpen() && row.ConnectionID == req.ConnectionID &&
			slices.Contains(req.ObjectIDs, row.ObjectID) {
			out = append(out, cloneFinding(row))
		}
	}
	return out, nil
}

type fakeSource struct {
	mu        sync.Mutex
	scope     int64
	records   []*accountingsync.AccountingSyncRecord
	states    map[pulid.ID]*repositories.AccountingDriftState
	balances  []*repositories.AccountingDriftBalanceLine
	pending   []pulid.ID
	listCalls []repositories.ListAccountingDriftRecordsRequest
}

func (f *fakeSource) ScopeStart(context.Context, pagination.TenantInfo) (int64, error) {
	return f.scope, nil
}

func (f *fakeSource) ListRecords(
	_ context.Context,
	req *repositories.ListAccountingDriftRecordsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls = append(f.listCalls, *req)
	out := []*accountingsync.AccountingSyncRecord{}
	for _, record := range f.records {
		if !req.AfterID.IsNil() && record.ID.String() <= req.AfterID.String() {
			continue
		}
		if len(out) == req.Limit {
			break
		}
		copied := *record
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeSource) ListStates(
	_ context.Context,
	req *repositories.ListAccountingDriftStatesRequest,
) ([]*repositories.AccountingDriftState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*repositories.AccountingDriftState{}
	for _, id := range req.ObjectIDs {
		if state, ok := f.states[id]; ok {
			copied := *state
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeSource) ListBalances(
	_ context.Context,
	req *repositories.ListAccountingDriftBalancesRequest,
) ([]*repositories.AccountingDriftBalanceLine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*repositories.AccountingDriftBalanceLine{}
	customers := 0
	var last pulid.ID
	for _, line := range f.balances {
		if !req.AfterCustomerID.IsNil() && line.CustomerID.String() <= req.AfterCustomerID.String() {
			continue
		}
		if line.CustomerID != last {
			if customers == req.Customers {
				break
			}
			customers++
			last = line.CustomerID
		}
		copied := *line
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeSource) ListPendingCustomers(
	_ context.Context,
	req *repositories.ListAccountingDriftPendingCustomersRequest,
) ([]pulid.ID, error) {
	out := []pulid.ID{}
	for _, id := range f.pending {
		if slices.Contains(req.CustomerIDs, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

func decimalOf(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func balanceOf(value string) *decimal.Decimal {
	d := decimal.RequireFromString(value)
	return &d
}
