package accountingmappingservice

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type fakeConnections struct {
	accountingconnectionports.AccountingConnectionRepository
	mu      sync.Mutex
	rows    map[pulid.ID]*accountingsync.AccountingConnection
	refresh []accountingconnectionports.MarkAccountingReferenceRefreshRequest
}

func newFakeConnections(conns ...*accountingsync.AccountingConnection) *fakeConnections {
	f := &fakeConnections{rows: map[pulid.ID]*accountingsync.AccountingConnection{}}
	for _, conn := range conns {
		f.rows[conn.ID] = conn
	}
	return f
}

func (f *fakeConnections) GetByType(
	_ context.Context,
	req accountingconnectionports.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.OrganizationID == req.TenantInfo.OrgID &&
			row.BusinessUnitID == req.TenantInfo.BuID &&
			row.IntegrationType == req.IntegrationType {
			out := *row
			return &out, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) GetByID(
	_ context.Context,
	req accountingconnectionports.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok || row.OrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("Accounting connection not found")
	}
	out := *row
	return &out, nil
}

func (f *fakeConnections) Update(
	_ context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entity.Version++
	stored := *entity
	f.rows[entity.ID] = &stored
	out := stored
	return &out, nil
}

func (f *fakeConnections) SaveDriftCheck(
	context.Context,
	*accountingconnectionports.SaveAccountingDriftCheckRequest,
) error {
	return nil
}

func (f *fakeConnections) SaveChangeFeed(
	context.Context,
	*accountingconnectionports.SaveAccountingChangeFeedRequest,
) (bool, error) {
	return false, nil
}

func (f *fakeConnections) MarkReferenceRefresh(
	_ context.Context,
	req accountingconnectionports.MarkAccountingReferenceRefreshRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refresh = append(f.refresh, req)
	return nil
}

type fakeReferences struct {
	mu      sync.Mutex
	rows    []*accountingsync.AccountingReferenceObject
	seen    map[string]int64
	removed []accountingsync.ReferenceKind
}

func newFakeReferences(refs ...*accountingsync.AccountingReferenceObject) *fakeReferences {
	return &fakeReferences{rows: refs, seen: map[string]int64{}}
}

func refKey(kind accountingsync.ReferenceKind, externalID string) string {
	return string(kind) + "|" + externalID
}

func (f *fakeReferences) Upsert(
	_ context.Context,
	req *accountingmappingports.UpsertAccountingReferenceObjectsRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, obj := range req.Objects {
		f.seen[refKey(obj.Kind, obj.ExternalID)] = req.SeenAt
		idx := slices.IndexFunc(f.rows, func(r *accountingsync.AccountingReferenceObject) bool {
			return r.Kind == obj.Kind && r.ExternalID == obj.ExternalID
		})
		copied := *obj
		copied.ConnectionID = req.ConnectionID
		if idx >= 0 {
			f.rows[idx] = &copied
			continue
		}
		f.rows = append(f.rows, &copied)
	}
	return nil
}

func (f *fakeReferences) MarkRemoved(
	context.Context,
	*accountingmappingports.MarkAccountingReferencesRemovedRequest,
) (int64, error) {
	return 0, nil
}

func (f *fakeReferences) MarkRemovedUnseen(
	_ context.Context,
	req *accountingmappingports.MarkAccountingReferenceRemovedRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, req.Kind)
	var count int64
	for _, row := range f.rows {
		if row.Kind == req.Kind && f.seen[refKey(row.Kind, row.ExternalID)] < req.SeenBefore {
			count++
		}
	}
	return count, nil
}

func (f *fakeReferences) ListByKind(
	_ context.Context,
	req *accountingmappingports.ListAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingReferenceObject, 0)
	for _, row := range f.rows {
		if row.Kind == req.Kind && (!req.UsableOnly || row.Usable()) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeReferences) Search(
	_ context.Context,
	req *accountingmappingports.SearchAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingReferenceObject, 0)
	for _, row := range f.rows {
		if row.Kind == req.Kind &&
			strings.Contains(strings.ToLower(row.Name), strings.ToLower(req.Query)) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeReferences) GetByExternalIDs(
	_ context.Context,
	req *accountingmappingports.GetAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingReferenceObject, 0, len(req.ExternalIDs))
	for _, row := range f.rows {
		if row.Kind == req.Kind && slices.Contains(req.ExternalIDs, row.ExternalID) {
			out = append(out, row)
		}
	}
	return out, nil
}

type fakeMappings struct {
	ownerOperators []*worker.Worker
	mu             sync.Mutex
	rows           []*accountingsync.AccountingMapping
}

func cloneMapping(row *accountingsync.AccountingMapping) *accountingsync.AccountingMapping {
	out := *row
	out.Signals.Candidates = slices.Clone(row.Signals.Candidates)
	out.Signals.Matchers = slices.Clone(row.Signals.Matchers)
	out.Signals.RejectedExternalIDs = slices.Clone(row.Signals.RejectedExternalIDs)
	return &out
}

func (f *fakeMappings) find(id pulid.ID) int {
	return slices.IndexFunc(
		f.rows,
		func(r *accountingsync.AccountingMapping) bool { return r.ID == id },
	)
}

func (f *fakeMappings) byTarget(
	targetType accountingsync.MappingTargetType,
	objectID pulid.ID,
	key string,
) *accountingsync.AccountingMapping {
	for _, row := range f.rows {
		if row.TargetType == targetType && row.TrenovaObjectID == objectID &&
			row.TrenovaKey == key {
			return row
		}
	}
	return nil
}

func (f *fakeMappings) ListByConnection(
	_ context.Context,
	req *accountingmappingports.ListAccountingMappingsQuery,
) ([]*accountingsync.AccountingMapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingMapping, 0, len(f.rows))
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID {
			out = append(out, cloneMapping(row))
		}
	}
	return out, nil
}

func (f *fakeMappings) ListConnection(
	_ context.Context,
	req *accountingmappingports.ListAccountingMappingsConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]*accountingsync.AccountingMapping, 0)
	for _, row := range f.rows {
		if row.ConnectionID != req.ConnectionID {
			continue
		}
		if req.RequiredOnly && !row.IsRequired() {
			continue
		}
		if len(req.States) > 0 && !slices.Contains(req.States, row.State) {
			continue
		}
		items = append(items, cloneMapping(row))
	}
	return &pagination.CursorListResult[*accountingsync.AccountingMapping]{Items: items}, nil
}

func (f *fakeMappings) GetByID(
	_ context.Context,
	req accountingmappingports.GetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.find(req.ID)
	if idx < 0 || f.rows[idx].OrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("Accounting mapping not found")
	}
	return cloneMapping(f.rows[idx]), nil
}

func (f *fakeMappings) GetByIDs(
	_ context.Context,
	req accountingmappingports.GetAccountingMappingsByIDsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingMapping, 0, len(req.IDs))
	for _, row := range f.rows {
		if row.OrganizationID == req.TenantInfo.OrgID && slices.Contains(req.IDs, row.ID) {
			out = append(out, cloneMapping(row))
		}
	}
	return out, nil
}

func (f *fakeMappings) GetByTarget(
	_ context.Context,
	req *accountingmappingports.GetAccountingMappingByTargetRequest,
) (*accountingsync.AccountingMapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row := f.byTarget(req.TargetType, req.TrenovaObjectID, req.TrenovaKey)
	if row == nil || row.ConnectionID != req.ConnectionID {
		return nil, errortypes.NewNotFoundError("Accounting mapping not found")
	}
	return cloneMapping(row), nil
}

func (f *fakeMappings) CreateMissing(
	_ context.Context,
	entities []*accountingsync.AccountingMapping,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var created int64
	for _, entity := range entities {
		if f.byTarget(entity.TargetType, entity.TrenovaObjectID, entity.TrenovaKey) != nil {
			continue
		}
		row := cloneMapping(entity)
		row.ID = pulid.MustNew("acctm_")
		f.rows = append(f.rows, row)
		created++
	}
	return created, nil
}

func (f *fakeMappings) ApplyScoring(
	_ context.Context,
	entities []*accountingsync.AccountingMapping,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var updated int64
	for _, entity := range entities {
		idx := f.find(entity.ID)
		if idx < 0 || f.rows[idx].Version != entity.Version ||
			f.rows[idx].State == accountingsync.MappingStateConfirmed {
			continue
		}
		row := cloneMapping(entity)
		row.Version++
		f.rows[idx] = row
		updated++
	}
	return updated, nil
}

func (f *fakeMappings) UpdateLabels(
	_ context.Context,
	entities []*accountingsync.AccountingMapping,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, entity := range entities {
		if idx := f.find(entity.ID); idx >= 0 {
			f.rows[idx].TargetLabel = entity.TargetLabel
		}
	}
	return nil
}

func (f *fakeMappings) Update(
	_ context.Context,
	entity *accountingsync.AccountingMapping,
) (*accountingsync.AccountingMapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.find(entity.ID)
	if idx < 0 {
		return nil, errortypes.NewNotFoundError("Accounting mapping not found")
	}
	if f.rows[idx].Version != entity.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Version mismatch",
		)
	}
	row := cloneMapping(entity)
	row.Version++
	f.rows[idx] = row
	return cloneMapping(row), nil
}

func (f *fakeMappings) ListOwnerOperators(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*worker.Worker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.ownerOperators), nil
}

func (f *fakeMappings) CountByState(
	_ context.Context,
	_ pagination.TenantInfo,
	connectionID pulid.ID,
) ([]accountingmappingports.AccountingMappingCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := make(map[[2]string]int)
	for _, row := range f.rows {
		if row.ConnectionID == connectionID {
			counts[[2]string{string(row.TargetType), string(row.State)}]++
		}
	}
	out := make([]accountingmappingports.AccountingMappingCount, 0, len(counts))
	for key, count := range counts {
		out = append(out, accountingmappingports.AccountingMappingCount{
			TargetType: accountingsync.MappingTargetType(key[0]),
			State:      accountingsync.MappingState(key[1]),
			Count:      count,
		})
	}
	return out, nil
}

func (f *fakeMappings) get(id pulid.ID) *accountingsync.AccountingMapping {
	f.mu.Lock()
	defer f.mu.Unlock()
	if idx := f.find(id); idx >= 0 {
		return cloneMapping(f.rows[idx])
	}
	return nil
}

type fakeConnector struct {
	accountingconnectorports.AccountingConnector
	mu         sync.Mutex
	pages      map[accountingsync.ReferenceKind][][]*accountingsync.AccountingReferenceObject
	listErr    error
	createErr  error
	duplicate  bool
	created    []*accountingconnectorports.AccountingCreateReferenceRequest
	createdObj *accountingsync.AccountingReferenceObject
	onCreate   func()
}

func (f *fakeConnector) IntegrationType() integration.Type { return integration.TypeQuickBooksOnline }

func (f *fakeConnector) MaxReferencePageSize() int { return 2 }

func (f *fakeConnector) ListReference(
	_ context.Context,
	req *accountingconnectorports.AccountingReferencePageRequest,
) (*accountingconnectorports.AccountingReferencePage, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	pages := f.pages[req.Kind]
	idx := (req.StartPosition - 1) / req.PageSize
	if idx >= len(pages) {
		return &accountingconnectorports.AccountingReferencePage{}, nil
	}
	next := 0
	if idx+1 < len(pages) {
		next = req.StartPosition + req.PageSize
	}
	return &accountingconnectorports.AccountingReferencePage{
		Objects:   pages[idx],
		NextStart: next,
	}, nil
}

func (f *fakeConnector) CreateReference(
	_ context.Context,
	req *accountingconnectorports.AccountingCreateReferenceRequest,
) (*accountingsync.AccountingReferenceObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, req)
	if f.onCreate != nil {
		f.onCreate()
	}
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createdObj, nil
}

func (f *fakeConnector) IsDuplicateName(err error) bool { return f.duplicate && err != nil }

func (f *fakeConnector) SanitizeName(_ accountingsync.ReferenceKind, name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, ":", " "))
}

type fakeConnectionService struct {
	accountingconnectionports.AccountingConnectionService
	session    *accountingconnectionports.AccountingSession
	sessionErr error
	failures   []error
}

func (f *fakeConnectionService) Session(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*accountingconnectionports.AccountingSession, error) {
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}
	return f.session, nil
}

func (f *fakeConnectionService) ReportCallFailure(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	cause error,
) accountingsync.ErrorCategory {
	f.failures = append(f.failures, cause)
	return accountingsync.ErrorCategoryTransient
}

type fakeAudit struct {
	auditports.AuditService
	mu      sync.Mutex
	entries []*auditports.LogActionParams
}

func (f *fakeAudit) LogAction(params *auditports.LogActionParams, _ ...auditports.LogOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, params)
	return nil
}

type fakeCompletion struct {
	llmports.CompletionService
	reply    string
	err      error
	requests []*llmports.StructuredCompletionRequest
}

func (f *fakeCompletion) CompleteStructured(
	_ context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	return &llmports.StructuredCompletionResult{Text: f.reply}, nil
}

type fakeRefresher struct {
	requested []pulid.ID
}

func (f *fakeRefresher) RequestReferenceRefresh(
	_ context.Context,
	_ pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	f.requested = append(f.requested, connectionID)
	return nil
}
