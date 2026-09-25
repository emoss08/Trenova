package accountingmappingservice

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type fakeConnections struct {
	repositories.AccountingConnectionRepository
	mu      sync.Mutex
	rows    map[pulid.ID]*accountingsync.AccountingConnection
	refresh []repositories.MarkAccountingReferenceRefreshRequest
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
	req repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.OrganizationID == req.TenantInfo.OrgID && row.BusinessUnitID == req.TenantInfo.BuID &&
			row.IntegrationType == req.IntegrationType {
			out := *row
			return &out, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) GetByID(
	_ context.Context,
	req repositories.GetAccountingConnectionByIDRequest,
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

func (f *fakeConnections) MarkReferenceRefresh(
	_ context.Context,
	req repositories.MarkAccountingReferenceRefreshRequest,
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
	req *repositories.UpsertAccountingReferenceObjectsRequest,
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

func (f *fakeReferences) MarkRemovedUnseen(
	_ context.Context,
	req *repositories.MarkAccountingReferenceRemovedRequest,
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
	req *repositories.ListAccountingReferenceObjectsRequest,
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
	req *repositories.SearchAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingReferenceObject, 0)
	for _, row := range f.rows {
		if row.Kind == req.Kind && strings.Contains(strings.ToLower(row.Name), strings.ToLower(req.Query)) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeReferences) GetByExternalIDs(
	_ context.Context,
	req *repositories.GetAccountingReferenceObjectsRequest,
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
	mu   sync.Mutex
	rows []*accountingsync.AccountingMapping
}

func cloneMapping(row *accountingsync.AccountingMapping) *accountingsync.AccountingMapping {
	out := *row
	out.Signals.Candidates = slices.Clone(row.Signals.Candidates)
	out.Signals.Matchers = slices.Clone(row.Signals.Matchers)
	out.Signals.RejectedExternalIDs = slices.Clone(row.Signals.RejectedExternalIDs)
	return &out
}

func (f *fakeMappings) find(id pulid.ID) int {
	return slices.IndexFunc(f.rows, func(r *accountingsync.AccountingMapping) bool { return r.ID == id })
}

func (f *fakeMappings) byTarget(
	targetType accountingsync.MappingTargetType,
	objectID pulid.ID,
	key string,
) *accountingsync.AccountingMapping {
	for _, row := range f.rows {
		if row.TargetType == targetType && row.TrenovaObjectID == objectID && row.TrenovaKey == key {
			return row
		}
	}
	return nil
}

func (f *fakeMappings) ListByConnection(
	_ context.Context,
	req *repositories.ListAccountingMappingsRequest,
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
	req *repositories.ListAccountingMappingsConnectionRequest,
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
	req repositories.GetAccountingMappingRequest,
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
	req repositories.GetAccountingMappingsByIDsRequest,
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
	req *repositories.GetAccountingMappingByTargetRequest,
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
		return nil, errortypes.NewBusinessError("Version mismatch")
	}
	row := cloneMapping(entity)
	row.Version++
	f.rows[idx] = row
	return cloneMapping(row), nil
}

func (f *fakeMappings) CountByState(
	_ context.Context,
	_ pagination.TenantInfo,
	connectionID pulid.ID,
) ([]repositories.AccountingMappingCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := make(map[[2]string]int)
	for _, row := range f.rows {
		if row.ConnectionID == connectionID {
			counts[[2]string{string(row.TargetType), string(row.State)}]++
		}
	}
	out := make([]repositories.AccountingMappingCount, 0, len(counts))
	for key, count := range counts {
		out = append(out, repositories.AccountingMappingCount{
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
	services.AccountingConnector
	mu         sync.Mutex
	pages      map[accountingsync.ReferenceKind][][]*accountingsync.AccountingReferenceObject
	listErr    error
	createErr  error
	duplicate  bool
	created    []*services.AccountingCreateReferenceRequest
	createdObj *accountingsync.AccountingReferenceObject
}

func (f *fakeConnector) IntegrationType() integration.Type { return integration.TypeQuickBooksOnline }

func (f *fakeConnector) MaxReferencePageSize() int { return 2 }

func (f *fakeConnector) ListReference(
	_ context.Context,
	req *services.AccountingReferencePageRequest,
) (*services.AccountingReferencePage, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	pages := f.pages[req.Kind]
	idx := (req.StartPosition - 1) / req.PageSize
	if idx >= len(pages) {
		return &services.AccountingReferencePage{}, nil
	}
	next := 0
	if idx+1 < len(pages) {
		next = req.StartPosition + req.PageSize
	}
	return &services.AccountingReferencePage{Objects: pages[idx], NextStart: next}, nil
}

func (f *fakeConnector) CreateReference(
	_ context.Context,
	req *services.AccountingCreateReferenceRequest,
) (*accountingsync.AccountingReferenceObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, req)
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
	services.AccountingConnectionService
	session    *services.AccountingSession
	sessionErr error
	failures   []error
}

func (f *fakeConnectionService) Session(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*services.AccountingSession, error) {
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

type fakeCompletion struct {
	services.CompletionService
	reply    string
	err      error
	requests []*services.StructuredCompletionRequest
}

func (f *fakeCompletion) CompleteStructured(
	_ context.Context,
	req *services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	return &services.StructuredCompletionResult{Text: f.reply}, nil
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
