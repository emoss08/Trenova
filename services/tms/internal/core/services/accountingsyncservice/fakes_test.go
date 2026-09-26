package accountingsyncservice

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func sameTenant(orgID, buID pulid.ID, tenantInfo pagination.TenantInfo) bool {
	return orgID == tenantInfo.OrgID && buID == tenantInfo.BuID
}

func cloneConnection(conn *accountingsync.AccountingConnection) *accountingsync.AccountingConnection {
	out := *conn
	return &out
}

type fakeConnections struct {
	repositories.AccountingConnectionRepository

	mu   sync.Mutex
	rows map[pulid.ID]*accountingsync.AccountingConnection
}

func newFakeConnections() *fakeConnections {
	return &fakeConnections{rows: map[pulid.ID]*accountingsync.AccountingConnection{}}
}

func (f *fakeConnections) put(conn *accountingsync.AccountingConnection) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[conn.ID] = cloneConnection(conn)
}

func (f *fakeConnections) get(id pulid.ID) *accountingsync.AccountingConnection {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneConnection(f.rows[id])
}

func (f *fakeConnections) GetByType(
	_ context.Context,
	req repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) &&
			row.IntegrationType == req.IntegrationType {
			return cloneConnection(row), nil
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
	if row, ok := f.rows[req.ID]; ok &&
		sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) {
		return cloneConnection(row), nil
	}
	return nil, errortypes.NewNotFoundError("Accounting connection not found")
}

func (f *fakeConnections) ListByTenant(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingConnection, 0, len(f.rows))
	for _, row := range f.rows {
		if sameTenant(row.OrganizationID, row.BusinessUnitID, tenantInfo) {
			out = append(out, cloneConnection(row))
		}
	}
	slices.SortFunc(out, func(a, b *accountingsync.AccountingConnection) int {
		return cmp.Compare(a.ID.String(), b.ID.String())
	})
	return out, nil
}

func (f *fakeConnections) Update(
	_ context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.rows[entity.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting connection not found")
	}
	if stored.Version != entity.Version {
		return nil, versionMismatch()
	}
	entity.Version++
	f.rows[entity.ID] = cloneConnection(entity)
	return cloneConnection(entity), nil
}

func versionMismatch() error {
	return errortypes.NewValidationError(
		"version",
		errortypes.ErrVersionMismatch,
		"The record was changed by someone else",
	)
}

func cloneRecord(record *accountingsync.AccountingSyncRecord) *accountingsync.AccountingSyncRecord {
	out := *record
	out.ExternalRefs = maps.Clone(record.ExternalRefs)
	out.MappingIDs = slices.Clone(record.MappingIDs)
	if record.Payload != nil {
		out.Payload = maps.Clone(record.Payload)
	}
	if record.NextAttemptAt != nil {
		next := *record.NextAttemptAt
		out.NextAttemptAt = &next
	}
	if record.LeaseExpiresAt != nil {
		lease := *record.LeaseExpiresAt
		out.LeaseExpiresAt = &lease
	}
	return &out
}

type candidateRow struct {
	candidate repositories.AccountingSyncCandidate
	orgID     pulid.ID
	buID      pulid.ID
}

type fakeRecords struct {
	repositories.AccountingSyncRecordRepository

	mu              sync.Mutex
	rows            []*accountingsync.AccountingSyncRecord
	attempts        []*accountingsync.AccountingSyncAttempt
	candidates      []candidateRow
	attention       []repositories.AccountingSyncAttentionGroup
	purgeRounds     []repositories.PurgeAccountingSyncHistoryResult
	purgeRequests   []repositories.PurgeAccountingSyncHistoryRequest
	candidateCalls  []repositories.ListAccountingSyncCandidatesRequest
	byObjectsCalls  []repositories.ListAccountingSyncRecordsByObjectsRequest
	connectionCalls []repositories.ListAccountingSyncRecordsConnectionRequest
	dueCalls        []repositories.ListDueAccountingSyncConnectionsRequest
	attentionCalls  int
	finishErr       map[pulid.ID]error
	onCandidates    func()
}

func newFakeRecords() *fakeRecords {
	return &fakeRecords{finishErr: map[pulid.ID]error{}}
}

func (f *fakeRecords) indexOf(id pulid.ID) int {
	return slices.IndexFunc(f.rows, func(row *accountingsync.AccountingSyncRecord) bool {
		return row.ID == id
	})
}

func (f *fakeRecords) all() []*accountingsync.AccountingSyncRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*accountingsync.AccountingSyncRecord, 0, len(f.rows))
	for _, row := range f.rows {
		out = append(out, cloneRecord(row))
	}
	return out
}

func (f *fakeRecords) find(
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
	operation accountingsync.SyncOperation,
) []*accountingsync.AccountingSyncRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if row.ObjectType == objectType && row.ObjectID == objectID && row.Operation == operation {
			out = append(out, cloneRecord(row))
		}
	}
	return out
}

func (f *fakeRecords) get(id pulid.ID) *accountingsync.AccountingSyncRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.indexOf(id)
	if idx < 0 {
		return nil
	}
	return cloneRecord(f.rows[idx])
}

func (f *fakeRecords) put(record *accountingsync.AccountingSyncRecord) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if idx := f.indexOf(record.ID); idx >= 0 {
		f.rows[idx] = cloneRecord(record)
		return
	}
	f.rows = append(f.rows, cloneRecord(record))
}

func (f *fakeRecords) makeDue(id pulid.ID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.indexOf(id)
	past := time.Now().Add(-time.Second).Unix()
	f.rows[idx].NextAttemptAt = &past
}

func (f *fakeRecords) postpone(id pulid.ID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	later := time.Now().Add(time.Hour).Unix()
	f.rows[f.indexOf(id)].NextAttemptAt = &later
}

func (f *fakeRecords) attemptsFor(id pulid.ID) []*accountingsync.AccountingSyncAttempt {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingSyncAttempt{}
	for _, attempt := range f.attempts {
		if attempt.SyncRecordID == id {
			out = append(out, attempt)
		}
	}
	return out
}

func (f *fakeRecords) addCandidate(
	tenantInfo pagination.TenantInfo,
	candidate repositories.AccountingSyncCandidate,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.candidates = append(f.candidates, candidateRow{
		candidate: candidate,
		orgID:     tenantInfo.OrgID,
		buID:      tenantInfo.BuID,
	})
}

func (f *fakeRecords) Enqueue(
	_ context.Context,
	records []*accountingsync.AccountingSyncRecord,
) (*repositories.EnqueueAccountingSyncRecordsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := &repositories.EnqueueAccountingSyncRecordsResult{
		Inserted: make([]*accountingsync.AccountingSyncRecord, 0, len(records)),
	}
	for _, record := range records {
		duplicate := slices.ContainsFunc(f.rows, func(row *accountingsync.AccountingSyncRecord) bool {
			return row.OrganizationID == record.OrganizationID &&
				row.BusinessUnitID == record.BusinessUnitID &&
				row.ConnectionID == record.ConnectionID &&
				row.IdempotencyKey == record.IdempotencyKey
		})
		if duplicate {
			result.Existing++
			continue
		}
		f.rows = append(f.rows, cloneRecord(record))
		result.Inserted = append(result.Inserted, record)
	}
	return result, nil
}

func (f *fakeRecords) GetByID(
	_ context.Context,
	req repositories.GetAccountingSyncRecordRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.indexOf(req.ID)
	if idx < 0 || !sameTenant(f.rows[idx].OrganizationID, f.rows[idx].BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("Accounting sync record not found")
	}
	return cloneRecord(f.rows[idx]), nil
}

func (f *fakeRecords) ListConnection(
	_ context.Context,
	req *repositories.ListAccountingSyncRecordsConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connectionCalls = append(f.connectionCalls, *req)
	items := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID &&
			sameTenant(row.OrganizationID, row.BusinessUnitID, req.Filter.TenantInfo) {
			items = append(items, cloneRecord(row))
		}
	}
	return &pagination.CursorListResult[*accountingsync.AccountingSyncRecord]{Items: items}, nil
}

func (f *fakeRecords) ListByExternalIDs(
	_ context.Context,
	req *repositories.ListAccountingSyncRecordsByExternalIDsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if !sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) ||
			row.ConnectionID != req.ConnectionID || row.ExternalID == "" ||
			!slices.Contains(req.ExternalIDs, row.ExternalID) {
			continue
		}
		if len(req.ObjectTypes) > 0 && !slices.Contains(req.ObjectTypes, row.ObjectType) {
			continue
		}
		out = append(out, cloneRecord(row))
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
		if !sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) ||
			row.ConnectionID != req.ConnectionID ||
			row.Status != accountingsync.SyncStatusInFlight {
			continue
		}
		if len(req.ObjectTypes) > 0 && !slices.Contains(req.ObjectTypes, row.ObjectType) {
			continue
		}
		count++
	}
	return count, nil
}

func (f *fakeRecords) ListByObjects(
	_ context.Context,
	req *repositories.ListAccountingSyncRecordsByObjectsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byObjectsCalls = append(f.byObjectsCalls, *req)
	out := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if !sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) ||
			!slices.Contains(req.ObjectIDs, row.ObjectID) {
			continue
		}
		if !req.ConnectionID.IsNil() && row.ConnectionID != req.ConnectionID {
			continue
		}
		if len(req.ObjectTypes) > 0 && !slices.Contains(req.ObjectTypes, row.ObjectType) {
			continue
		}
		out = append(out, cloneRecord(row))
	}
	slices.SortStableFunc(out, func(a, b *accountingsync.AccountingSyncRecord) int {
		return cmp.Compare(a.QueuedAt, b.QueuedAt)
	})
	return out, nil
}

func due(row *accountingsync.AccountingSyncRecord, now int64) bool {
	if row.Status.Dispatchable() {
		return row.NextAttemptAt != nil && *row.NextAttemptAt <= now
	}
	return row.Status == accountingsync.SyncStatusInFlight &&
		row.LeaseExpiresAt != nil && *row.LeaseExpiresAt <= now
}

func (f *fakeRecords) Claim(
	_ context.Context,
	req *repositories.ClaimAccountingSyncRecordsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	picked := []*accountingsync.AccountingSyncRecord{}
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID &&
			sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) &&
			due(row, req.Now) {
			picked = append(picked, row)
		}
	}
	slices.SortStableFunc(picked, func(a, b *accountingsync.AccountingSyncRecord) int {
		return cmp.Or(
			cmp.Compare(a.ObjectType.DispatchRank(), b.ObjectType.DispatchRank()),
			cmp.Compare(a.QueuedAt, b.QueuedAt),
		)
	})
	if req.Limit > 0 && len(picked) > req.Limit {
		picked = picked[:req.Limit]
	}
	lease := req.Lease
	if lease <= 0 {
		lease = time.Minute
	}
	out := make([]*accountingsync.AccountingSyncRecord, 0, len(picked))
	for _, row := range picked {
		leaseAt := req.Now + int64(lease/time.Second)
		started := req.Now
		row.Status = accountingsync.SyncStatusInFlight
		row.LeaseExpiresAt = &leaseAt
		row.AttemptCount++
		row.StartedAt = &started
		row.Version++
		out = append(out, cloneRecord(row))
	}
	return out, nil
}

func (f *fakeRecords) Finish(
	_ context.Context,
	req *repositories.FinishAccountingSyncRecordRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.finishErr[req.Record.ID]; err != nil {
		return err
	}
	idx := f.indexOf(req.Record.ID)
	if idx < 0 {
		return repositories.ErrAccountingSyncLeaseLost
	}
	stored := f.rows[idx]
	if stored.Version != req.Record.Version || stored.Status != accountingsync.SyncStatusInFlight {
		return repositories.ErrAccountingSyncLeaseLost
	}
	req.Record.Version++
	f.rows[idx] = cloneRecord(req.Record)
	if req.Attempt != nil {
		f.attempts = append(f.attempts, req.Attempt)
	}
	return nil
}

func (f *fakeRecords) Update(
	_ context.Context,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsync.AccountingSyncRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.indexOf(record.ID)
	if idx < 0 {
		return nil, errortypes.NewNotFoundError("Accounting sync record not found")
	}
	if f.rows[idx].Version != record.Version {
		return nil, versionMismatch()
	}
	record.Version++
	f.rows[idx] = cloneRecord(record)
	return record, nil
}

func (f *fakeRecords) CountByStatus(
	_ context.Context,
	req repositories.AccountingSyncConnectionRequest,
) ([]repositories.AccountingSyncStatusCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := map[accountingsync.SyncStatus]int{}
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID {
			counts[row.Status]++
		}
	}
	out := make([]repositories.AccountingSyncStatusCount, 0, len(counts))
	for _, status := range accountingsync.AllSyncStatuses() {
		if counts[status] > 0 {
			out = append(out, repositories.AccountingSyncStatusCount{Status: status, Count: counts[status]})
		}
	}
	return out, nil
}

func (f *fakeRecords) ListAttention(
	_ context.Context,
	_ repositories.ListAccountingSyncAttentionRequest,
) ([]repositories.AccountingSyncAttentionGroup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attentionCalls++
	return slices.Clone(f.attention), nil
}

func (f *fakeRecords) Requeue(
	_ context.Context,
	req *repositories.RequeueAccountingSyncRecordsRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, row := range f.rows {
		if row.ConnectionID != req.ConnectionID || !row.Status.Retryable() {
			continue
		}
		if len(req.ErrorCategories) > 0 && !slices.Contains(req.ErrorCategories, row.ErrorCategory) {
			continue
		}
		if len(req.IDs) > 0 && !slices.Contains(req.IDs, row.ID) {
			continue
		}
		at := req.At
		row.Status = accountingsync.SyncStatusQueued
		row.AttemptCount = 0
		row.NextAttemptAt = &at
		row.LeaseExpiresAt = nil
		row.Version++
		count++
	}
	return count, nil
}

func (f *fakeRecords) Release(
	_ context.Context,
	req *repositories.ReleaseAccountingSyncRecordsRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, row := range f.rows {
		if row.ConnectionID != req.ConnectionID ||
			row.Status != accountingsync.SyncStatusAwaitingApproval {
			continue
		}
		if len(req.IDs) > 0 && !slices.Contains(req.IDs, row.ID) {
			continue
		}
		at := req.At
		row.Status = accountingsync.SyncStatusQueued
		row.ReleasedByID = req.ActorID
		row.NextAttemptAt = &at
		row.Version++
		count++
	}
	return count, nil
}

func (f *fakeRecords) SupersedeOlder(
	_ context.Context,
	req *repositories.SupersedeAccountingSyncRecordsRequest,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, row := range f.rows {
		if row.ConnectionID != req.ConnectionID || row.ObjectType != req.ObjectType ||
			row.ObjectID != req.ObjectID || row.Operation != req.Operation ||
			row.Revision >= req.BeforeRevision || row.Status.IsFinal() ||
			row.Status == accountingsync.SyncStatusInFlight {
			continue
		}
		row.Status = accountingsync.SyncStatusSuperseded
		row.NextAttemptAt = nil
		row.LeaseExpiresAt = nil
		row.Version++
		count++
	}
	return count, nil
}

func (f *fakeRecords) ListCandidates(
	_ context.Context,
	req *repositories.ListAccountingSyncCandidatesRequest,
) ([]repositories.AccountingSyncCandidate, error) {
	f.mu.Lock()
	f.candidateCalls = append(f.candidateCalls, *req)
	out := []repositories.AccountingSyncCandidate{}
	for _, row := range f.candidates {
		cand := row.candidate
		if !sameTenant(row.orgID, row.buID, req.TenantInfo) || cand.ObjectType != req.ObjectType ||
			cand.Operation != req.Operation || cand.DocumentDate < req.DatedFrom {
			continue
		}
		if req.PostedFrom != nil && cand.PostedAt < *req.PostedFrom {
			continue
		}
		if req.PostedBefore != nil && cand.PostedAt >= *req.PostedBefore {
			continue
		}
		if (req.AfterAt > 0 || !req.AfterID.IsNil()) &&
			cmp.Or(
				cmp.Compare(cand.PostedAt, req.AfterAt),
				cmp.Compare(cand.ObjectID.String(), req.AfterID.String()),
			) <= 0 {
			continue
		}
		recorded := slices.ContainsFunc(f.rows, func(record *accountingsync.AccountingSyncRecord) bool {
			return record.ConnectionID == req.ConnectionID && record.ObjectType == cand.ObjectType &&
				record.ObjectID == cand.ObjectID && record.Operation == cand.Operation
		})
		if !recorded {
			out = append(out, cand)
		}
	}
	slices.SortFunc(out, func(a, b repositories.AccountingSyncCandidate) int {
		return cmp.Or(cmp.Compare(a.PostedAt, b.PostedAt), cmp.Compare(a.ObjectID.String(), b.ObjectID.String()))
	})
	if req.Limit > 0 && len(out) > req.Limit {
		out = out[:req.Limit]
	}
	hook := f.onCandidates
	f.mu.Unlock()
	if hook != nil {
		hook()
	}
	return out, nil
}

func (f *fakeRecords) ListDueConnections(
	_ context.Context,
	req repositories.ListDueAccountingSyncConnectionsRequest,
) ([]repositories.AccountingSyncDueConnection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dueCalls = append(f.dueCalls, req)
	return []repositories.AccountingSyncDueConnection{}, nil
}

func (f *fakeRecords) PurgeHistory(
	_ context.Context,
	req repositories.PurgeAccountingSyncHistoryRequest,
) (*repositories.PurgeAccountingSyncHistoryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.purgeRequests = append(f.purgeRequests, req)
	if len(f.purgeRounds) == 0 {
		return &repositories.PurgeAccountingSyncHistoryResult{}, nil
	}
	round := f.purgeRounds[0]
	f.purgeRounds = f.purgeRounds[1:]
	return &round, nil
}

func (f *fakeRecords) ListAttempts(
	_ context.Context,
	req repositories.ListAccountingSyncAttemptsRequest,
) ([]*accountingsync.AccountingSyncAttempt, error) {
	return f.attemptsFor(req.SyncRecordID), nil
}

func cloneBackfill(backfill *accountingsync.AccountingBackfill) *accountingsync.AccountingBackfill {
	out := *backfill
	out.ObjectTypes = slices.Clone(backfill.ObjectTypes)
	return &out
}

type fakeBackfills struct {
	mu   sync.Mutex
	rows map[pulid.ID]*accountingsync.AccountingBackfill
}

func newFakeBackfills() *fakeBackfills {
	return &fakeBackfills{rows: map[pulid.ID]*accountingsync.AccountingBackfill{}}
}

func (f *fakeBackfills) get(id pulid.ID) *accountingsync.AccountingBackfill {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneBackfill(f.rows[id])
}

func (f *fakeBackfills) mutate(id pulid.ID, fn func(*accountingsync.AccountingBackfill)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f.rows[id])
	f.rows[id].Version++
}

func (f *fakeBackfills) Create(
	_ context.Context,
	entity *accountingsync.AccountingBackfill,
) (*accountingsync.AccountingBackfill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.ConnectionID == entity.ConnectionID && row.Status.InProgress() {
			return nil, repositories.ErrAccountingBackfillActive
		}
	}
	f.rows[entity.ID] = cloneBackfill(entity)
	return cloneBackfill(entity), nil
}

func (f *fakeBackfills) GetByID(
	_ context.Context,
	req repositories.GetAccountingBackfillRequest,
) (*accountingsync.AccountingBackfill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok || !sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("Backfill not found")
	}
	return cloneBackfill(row), nil
}

func (f *fakeBackfills) GetActive(
	_ context.Context,
	req repositories.AccountingSyncConnectionRequest,
) (*accountingsync.AccountingBackfill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID && row.Status.InProgress() {
			return cloneBackfill(row), nil
		}
	}
	return nil, errortypes.NewNotFoundError("Backfill not found")
}

func (f *fakeBackfills) ListByConnection(
	_ context.Context,
	req repositories.ListAccountingBackfillsRequest,
) ([]*accountingsync.AccountingBackfill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingBackfill{}
	for _, row := range f.rows {
		if row.ConnectionID == req.ConnectionID {
			out = append(out, cloneBackfill(row))
		}
	}
	return out, nil
}

func (f *fakeBackfills) Update(
	_ context.Context,
	entity *accountingsync.AccountingBackfill,
) (*accountingsync.AccountingBackfill, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.rows[entity.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Backfill not found")
	}
	if stored.Version != entity.Version {
		return nil, versionMismatch()
	}
	entity.Version++
	f.rows[entity.ID] = cloneBackfill(entity)
	return cloneBackfill(entity), nil
}

type vendorPartyCall struct {
	targetType accountingsync.MappingTargetType
	objectID   pulid.ID
}

type fakeMappingStore struct {
	mu          sync.Mutex
	rows        map[string]*accountingsync.AccountingMapping
	created     []*services.CreateAccountingReferenceRecordRequest
	createErr   error
	parties     map[pulid.ID]*services.AccountingPartyDraft
	vendorCalls []vendorPartyCall
}

func newFakeMappingStore() *fakeMappingStore {
	return &fakeMappingStore{
		rows:    map[string]*accountingsync.AccountingMapping{},
		parties: map[pulid.ID]*services.AccountingPartyDraft{},
	}
}

func mappingIdentity(
	connectionID pulid.ID,
	targetType accountingsync.MappingTargetType,
	objectID pulid.ID,
	key string,
) string {
	return mappingTarget{TargetType: targetType, ObjectID: objectID, Key: key}.identity() +
		"@" + connectionID.String()
}

func (f *fakeMappingStore) set(
	conn *accountingsync.AccountingConnection,
	target mappingTarget,
	state accountingsync.MappingState,
	externalID string,
) *accountingsync.AccountingMapping {
	f.mu.Lock()
	defer f.mu.Unlock()
	identity := mappingIdentity(conn.ID, target.TargetType, target.ObjectID, target.Key)
	row, ok := f.rows[identity]
	if !ok {
		row = &accountingsync.AccountingMapping{
			ID:              pulid.MustNew("acctm_"),
			OrganizationID:  conn.OrganizationID,
			BusinessUnitID:  conn.BusinessUnitID,
			ConnectionID:    conn.ID,
			TargetType:      target.TargetType,
			TrenovaObjectID: target.ObjectID,
			TrenovaKey:      target.Key,
			ProviderKind:    target.TargetType.ProviderKind(),
		}
		f.rows[identity] = row
	}
	row.TargetLabel = target.Label
	row.State = state
	row.ExternalID = externalID
	row.ExternalName = "QB " + target.Label
	out := *row
	return &out
}

func (f *fakeMappingStore) ensure(
	req *services.EnsureAccountingMappingRequest,
) *accountingsync.AccountingMapping {
	f.mu.Lock()
	defer f.mu.Unlock()
	identity := mappingIdentity(req.ConnectionID, req.TargetType, req.ObjectID, req.Key)
	row, ok := f.rows[identity]
	if !ok {
		row = &accountingsync.AccountingMapping{
			ID:              pulid.MustNew("acctm_"),
			OrganizationID:  req.TenantInfo.OrgID,
			BusinessUnitID:  req.TenantInfo.BuID,
			ConnectionID:    req.ConnectionID,
			TargetType:      req.TargetType,
			TrenovaObjectID: req.ObjectID,
			TrenovaKey:      req.Key,
			TargetLabel:     string(req.TargetType) + " " + req.Key,
			ProviderKind:    req.TargetType.ProviderKind(),
			State:           accountingsync.MappingStateUnmatched,
		}
		f.rows[identity] = row
	}
	out := *row
	return &out
}

type fakeMappingRepo struct {
	repositories.AccountingMappingRepository
	store *fakeMappingStore
}

func (f fakeMappingRepo) GetByTarget(
	_ context.Context,
	req *repositories.GetAccountingMappingByTargetRequest,
) (*accountingsync.AccountingMapping, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	identity := mappingIdentity(req.ConnectionID, req.TargetType, req.TrenovaObjectID, req.TrenovaKey)
	row, ok := f.store.rows[identity]
	if !ok {
		return nil, errortypes.NewNotFoundError("Mapping not found")
	}
	out := *row
	return &out, nil
}

type fakeMappingService struct {
	services.AccountingMappingService
	store *fakeMappingStore
}

func (f fakeMappingService) EnsureMapping(
	_ context.Context,
	req *services.EnsureAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	return f.store.ensure(req), nil
}

func (f fakeMappingService) CreateReferenceRecord(
	_ context.Context,
	req *services.CreateAccountingReferenceRecordRequest,
) (*accountingsync.AccountingMapping, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	f.store.created = append(f.store.created, req)
	if f.store.createErr != nil {
		return nil, f.store.createErr
	}
	for _, row := range f.store.rows {
		if row.ID == req.MappingID {
			row.State = accountingsync.MappingStateConfirmed
			row.ExternalID = "qb-created-" + row.TrenovaObjectID.String()
			row.ExternalName = row.TargetLabel
			row.Source = req.Source
			out := *row
			return &out, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Mapping not found")
}

func (f fakeMappingService) CustomerParty(
	_ context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
) (*services.AccountingPartyDraft, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	party, ok := f.store.parties[customerID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Customer not found")
	}
	out := *party
	return &out, nil
}

func (f fakeMappingService) VendorParty(
	_ context.Context,
	_ pagination.TenantInfo,
	targetType accountingsync.MappingTargetType,
	objectID pulid.ID,
) (*services.AccountingPartyDraft, error) {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()
	f.store.vendorCalls = append(f.store.vendorCalls, vendorPartyCall{
		targetType: targetType,
		objectID:   objectID,
	})
	party, ok := f.store.parties[objectID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Vendor not found")
	}
	out := *party
	return &out, nil
}

type reportedFailure struct {
	connectionID pulid.ID
	cause        error
}

type fakeConnService struct {
	services.AccountingConnectionService

	mu          sync.Mutex
	connections *fakeConnections
	writer      *fakeWriter
	accessToken string
	sessionErr  error
	reported    []reportedFailure
}

func (f *fakeConnService) Session(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*services.AccountingSession, error) {
	f.mu.Lock()
	sessionErr := f.sessionErr
	token := f.accessToken
	f.mu.Unlock()
	if sessionErr != nil {
		return nil, sessionErr
	}
	conn, err := f.connections.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         connectionID,
	})
	if err != nil {
		return nil, err
	}
	return &services.AccountingSession{Connection: conn, AccessToken: token, Connector: f.writer}, nil
}

func (f *fakeConnService) ReportCallFailure(
	_ context.Context,
	_ pagination.TenantInfo,
	connectionID pulid.ID,
	cause error,
) accountingsync.ErrorCategory {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reported = append(f.reported, reportedFailure{connectionID: connectionID, cause: cause})
	return accountingsync.ErrorCategoryUnauthorized
}

func (f *fakeConnService) reports() []reportedFailure {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reported)
}

type providerError struct {
	failure accountingsync.SyncError
}

func (e *providerError) Error() string { return e.failure.Message }

func providerFault(category accountingsync.SyncErrorCategory, message, resolution string) error {
	return &providerError{failure: accountingsync.SyncError{
		Category:   category,
		Code:       "QB-" + string(category),
		Message:    message,
		Resolution: resolution,
	}}
}

type writerCall struct {
	method string
	doc    any
}

type fakeWriter struct {
	services.AccountingConnector

	mu        sync.Mutex
	limits    services.AccountingDocumentLimits
	calls     []writerCall
	errs      map[string][]error
	partials  map[string]map[string]string
	found     map[string]string
	nextID    int
	unmatched bool
	extraRefs map[string]map[string]string
}

func newFakeWriter() *fakeWriter {
	return &fakeWriter{
		limits:    services.AccountingDocumentLimits{MaxDocNumberLength: 21},
		errs:      map[string][]error{},
		partials:  map[string]map[string]string{},
		found:     map[string]string{},
		extraRefs: map[string]map[string]string{},
	}
}

func (f *fakeWriter) failNext(method string, errs ...error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[method] = append(f.errs[method], errs...)
}

func (f *fakeWriter) callsTo(method string) []writerCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []writerCall{}
	for _, call := range f.calls {
		if call.method == method {
			out = append(out, call)
		}
	}
	return out
}

func (f *fakeWriter) methods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.calls))
	for _, call := range f.calls {
		out = append(out, call.method)
	}
	return out
}

func (f *fakeWriter) record(method string, doc any) (*services.AccountingDocumentResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, writerCall{method: method, doc: doc})
	if queued := f.errs[method]; len(queued) > 0 {
		f.errs[method] = queued[1:]
		var written *services.AccountingDocumentResult
		if refs := f.partials[method]; refs != nil {
			written = &services.AccountingDocumentResult{Refs: maps.Clone(refs)}
		}
		return written, queued[0]
	}
	f.nextID++
	id := "qb-" + method + "-" + strconv.Itoa(f.nextID)
	refs := map[string]string{accountingsync.ExternalRefDocument: id}
	maps.Copy(refs, f.extraRefs[method])
	return &services.AccountingDocumentResult{
		ExternalID: id,
		DocNumber:  "QB" + id,
		Refs:       refs,
	}, nil
}

func (f *fakeWriter) DocumentLimits() services.AccountingDocumentLimits {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.limits
}

func (f *fakeWriter) DocumentURL(kind accountingsync.SyncObjectType, externalID string) string {
	return "https://qbo.test/" + string(kind) + "/" + externalID
}

func (f *fakeWriter) ClassifyDocumentError(err error) *accountingsync.SyncError {
	f.mu.Lock()
	unmatched := f.unmatched
	f.mu.Unlock()
	if unmatched {
		return nil
	}
	var provider *providerError
	if errors.As(err, &provider) {
		out := provider.failure
		return &out
	}
	return nil
}

func (f *fakeWriter) UpsertCustomer(
	_ context.Context,
	doc *services.AccountingCustomerDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("UpsertCustomer", doc)
}

func (f *fakeWriter) CreateSalesDocument(
	_ context.Context,
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	sent := *doc
	sent.Refs = maps.Clone(doc.Refs)
	return f.record("CreateSalesDocument", &sent)
}

func (f *fakeWriter) VoidSalesDocument(
	_ context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	return f.record("VoidSalesDocument", ref)
}

func (f *fakeWriter) SavePayment(
	_ context.Context,
	doc *services.AccountingPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("SavePayment", doc)
}

func (f *fakeWriter) VoidPayment(
	_ context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	return f.record("VoidPayment", ref)
}

func (f *fakeWriter) CreateCreditApplication(
	_ context.Context,
	doc *services.AccountingCreditApplicationDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("CreateCreditApplication", doc)
}

func (f *fakeWriter) VoidCreditApplication(
	_ context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	return f.record("VoidCreditApplication", ref)
}

func (f *fakeWriter) UpsertVendor(
	_ context.Context,
	doc *services.AccountingVendorDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("UpsertVendor", doc)
}

func (f *fakeWriter) CreatePurchaseDocument(
	_ context.Context,
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("CreatePurchaseDocument", doc)
}

func (f *fakeWriter) UpdateSalesDocument(
	_ context.Context,
	doc *services.AccountingSalesDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("UpdateSalesDocument", doc)
}

func (f *fakeWriter) UpdatePurchaseDocument(
	_ context.Context,
	doc *services.AccountingPurchaseDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("UpdatePurchaseDocument", doc)
}

func (f *fakeWriter) UpdateBillPayment(
	_ context.Context,
	doc *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("UpdateBillPayment", doc)
}

func (f *fakeWriter) VoidPurchaseDocument(
	_ context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	return f.record("VoidPurchaseDocument", ref)
}

func (f *fakeWriter) CreateBillPayment(
	_ context.Context,
	doc *services.AccountingBillPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	return f.record("CreateBillPayment", doc)
}

func (f *fakeWriter) FindSalesDocument(
	_ context.Context,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, writerCall{method: "FindSalesDocument", doc: req})
	id, ok := f.found[req.DocNumber]
	if !ok {
		return nil, false, nil
	}
	return &services.AccountingDocumentResult{ExternalID: id, DocNumber: req.DocNumber}, true, nil
}

type fakeInvoices struct {
	repositories.InvoiceRepository

	mu   sync.Mutex
	rows map[pulid.ID]*invoice.Invoice
}

func (f *fakeInvoices) put(inv *invoice.Invoice) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[inv.ID] = inv
}

func (f *fakeInvoices) GetByID(
	_ context.Context,
	req repositories.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok || !sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("Invoice not found")
	}
	return row, nil
}

func (f *fakeInvoices) GetByIDs(
	_ context.Context,
	req repositories.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*invoice.Invoice{}
	for _, id := range req.InvoiceIDs {
		if row, ok := f.rows[id]; ok && sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) {
			out = append(out, row)
		}
	}
	return out, nil
}

type fakeAdjustments struct {
	repositories.InvoiceAdjustmentRepository

	mu   sync.Mutex
	rows map[pulid.ID]*invoiceadjustment.InvoiceAdjustment
}

func (f *fakeAdjustments) GetByID(
	_ context.Context,
	req repositories.GetInvoiceAdjustmentRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Adjustment not found")
	}
	return row, nil
}

type fakePayments struct {
	repositories.CustomerPaymentRepository

	mu           sync.Mutex
	rows         map[pulid.ID]*customerpayment.Payment
	applications map[pulid.ID]*customerpayment.CreditMemoApplication
}

func (f *fakePayments) GetByID(
	_ context.Context,
	req repositories.GetCustomerPaymentByIDRequest,
) (*customerpayment.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[req.ID]
	if !ok || !sameTenant(row.OrganizationID, row.BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("Payment not found")
	}
	return row, nil
}

func (f *fakePayments) GetCreditMemoApplicationByID(
	_ context.Context,
	req repositories.GetCreditMemoApplicationRequest,
) (*customerpayment.CreditMemoApplication, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.applications[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Credit application not found")
	}
	return row, nil
}

type fakeOrganizations struct {
	repositories.OrganizationRepository
	timezone string
}

func (f fakeOrganizations) GetByID(
	_ context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{ID: req.TenantInfo.OrgID, Timezone: f.timezone}, nil
}

type fakeAudit struct {
	services.AuditService

	mu       sync.Mutex
	entries  []*services.LogActionParams
	comments []string
}

func (f *fakeAudit) LogAction(params *services.LogActionParams, opts ...services.LogOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, params)
	entry := new(audit.Entry)
	for _, opt := range opts {
		if err := opt(entry); err != nil {
			return err
		}
	}
	f.comments = append(f.comments, entry.Comment)
	return nil
}

func (f *fakeAudit) lastComment() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.comments) == 0 {
		return ""
	}
	return f.comments[len(f.comments)-1]
}

func (f *fakeAudit) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.entries)
}

type fakeWatchtower struct {
	mu       sync.Mutex
	upserts  map[string]services.WatchtowerItemInput
	resolved map[string]bool
}

func newFakeWatchtower() *fakeWatchtower {
	return &fakeWatchtower{
		upserts:  map[string]services.WatchtowerItemInput{},
		resolved: map[string]bool{},
	}
}

func (f *fakeWatchtower) Upsert(_ context.Context, input services.WatchtowerItemInput) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.upserts[input.SourceID] = input
	delete(f.resolved, input.SourceID)
}

func (f *fakeWatchtower) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if kind != watchtower.SourceAccountingSync {
		panic("resolved with the wrong kind: " + string(kind))
	}
	delete(f.upserts, sourceID)
	f.resolved[sourceID] = true
}

func (f *fakeWatchtower) item(sourceID string) (services.WatchtowerItemInput, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.upserts[sourceID]
	return item, ok
}

type fakeDispatcher struct {
	mu         sync.Mutex
	kicks      []pulid.ID
	backfills  []*accountingsync.AccountingBackfill
	kickErr    error
	backfilled error
}

func (f *fakeDispatcher) Kick(_ context.Context, _ pagination.TenantInfo, connectionID pulid.ID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kicks = append(f.kicks, connectionID)
	return f.kickErr
}

func (f *fakeDispatcher) StartBackfill(_ context.Context, backfill *accountingsync.AccountingBackfill) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.backfills = append(f.backfills, cloneBackfill(backfill))
	return f.backfilled
}

func (f *fakeDispatcher) kicked() []pulid.ID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.kicks)
}

func (f *fakeDispatcher) started() []*accountingsync.AccountingBackfill {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.backfills)
}

type payableRow struct {
	settlement *repositories.PayableSettlement
	tenant     pagination.TenantInfo
}

type fakePayables struct {
	mu    sync.Mutex
	rows  map[pulid.ID]payableRow
	calls []repositories.GetPayableSettlementRequest
}

func newFakePayables() *fakePayables {
	return &fakePayables{rows: map[pulid.ID]payableRow{}}
}

func (f *fakePayables) put(
	tenantInfo pagination.TenantInfo,
	settlement *repositories.PayableSettlement,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[settlement.ID] = payableRow{settlement: settlement, tenant: tenantInfo}
}

func (f *fakePayables) requests() []repositories.GetPayableSettlementRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

func clonePayable(in *repositories.PayableSettlement) *repositories.PayableSettlement {
	out := *in
	out.Lines = slices.Clone(in.Lines)
	out.InvoiceNumbers = slices.Clone(in.InvoiceNumbers)
	if in.PostedAt != nil {
		posted := *in.PostedAt
		out.PostedAt = &posted
	}
	if in.PaidAt != nil {
		paid := *in.PaidAt
		out.PaidAt = &paid
	}
	return &out
}

func (f *fakePayables) GetSettlement(
	_ context.Context,
	req *repositories.GetPayableSettlementRequest,
) (*repositories.PayableSettlement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, *req)
	row, ok := f.rows[req.ID]
	if !ok || row.tenant != req.TenantInfo || row.settlement.Kind != req.Kind {
		return nil, errortypes.NewNotFoundError("Settlement not found")
	}
	return clonePayable(row.settlement), nil
}
