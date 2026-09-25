package accountingsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	SyncRequestIDPrefix       = "trn-"
	syncRequestIDHashChars    = 40
	MaxSyncAttempts           = 8
	SyncRetryBase             = 30 * time.Second
	SyncRetryCeiling          = 6 * time.Hour
	SyncAuthWait              = 15 * time.Minute
	maxSyncErrorMessage       = 2000
	maxSyncResolution         = 1000
	maxSyncSkipReason         = 500
	SyncPayloadRetention      = 90 * 24 * time.Hour
	SyncAttemptRetention      = 90 * 24 * time.Hour
	syncRecordIDPrefix        = "acctsr_"
	syncAttemptIDPrefix       = "acctsa_"
	backfillIDPrefix          = "acctbf_"
	ExternalRefDocument       = "document"
	ExternalRefApplication    = "application"
	ExternalRefShortPayPrefix = "shortPay:"
	ExternalRefDocumentType   = "documentType"
	ExternalRefURL            = "url"
)

var (
	ErrSyncRecordNotRetryable = errors.New(
		"only a blocked, dead-lettered or retrying record can be retried",
	)
	ErrSyncRecordNotSkippable  = errors.New("a record already sent or skipped cannot be skipped")
	ErrSyncRecordNotReleasable = errors.New("only a record waiting for approval can be released")
	ErrSkipReasonRequired      = errors.New("a reason is required to skip a record")
)

type AccountingSyncRecord struct {
	bun.BaseModel             `bun:"table:accounting_sync_records,alias:acctsr" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                     json:"-"`

	ID                pulid.ID          `json:"id"                bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID          `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ConnectionID      pulid.ID          `json:"connectionId"      bun:"connection_id,type:VARCHAR(100),notnull"`
	ObjectType        SyncObjectType    `json:"objectType"        bun:"object_type,type:VARCHAR(30),notnull"`
	ObjectID          pulid.ID          `json:"objectId"          bun:"object_id,type:VARCHAR(100),notnull"`
	ObjectNumber      string            `json:"objectNumber"      bun:"object_number,type:VARCHAR(100),nullzero"`
	Operation         SyncOperation     `json:"operation"         bun:"operation,type:VARCHAR(20),notnull"`
	SourceEvent       SyncSourceEvent   `json:"sourceEvent"       bun:"source_event,type:VARCHAR(40),notnull"`
	IdempotencyKey    string            `json:"idempotencyKey"    bun:"idempotency_key,type:VARCHAR(200),notnull"`
	RequestID         string            `json:"requestId"         bun:"request_id,type:VARCHAR(50),notnull"`
	Revision          int64             `json:"revision"          bun:"revision,type:BIGINT,notnull"`
	DocumentDate      *int64            `json:"documentDate"      bun:"document_date,type:BIGINT,nullzero"`
	DependsOnRecordID pulid.ID          `json:"dependsOnRecordId" bun:"depends_on_record_id,type:VARCHAR(100),nullzero"`
	Status            SyncStatus        `json:"status"            bun:"status,type:VARCHAR(20),notnull"`
	AttemptCount      int               `json:"attemptCount"      bun:"attempt_count,type:INTEGER,notnull"`
	NextAttemptAt     *int64            `json:"nextAttemptAt"     bun:"next_attempt_at,type:BIGINT,nullzero"`
	LeaseExpiresAt    *int64            `json:"-"                 bun:"lease_expires_at,type:BIGINT,nullzero"`
	ExternalID        string            `json:"externalId"        bun:"external_id,type:VARCHAR(100),nullzero"`
	ExternalDocNumber string            `json:"externalDocNumber" bun:"external_doc_number,type:VARCHAR(100),nullzero"`
	ExternalURL       string            `json:"externalUrl"       bun:"external_url,type:TEXT,nullzero"`
	ExternalRefs      map[string]string `json:"externalRefs"      bun:"external_refs,type:JSONB,notnull"`
	PayloadHash       string            `json:"payloadHash"       bun:"payload_hash,type:VARCHAR(64),nullzero"`
	Payload           map[string]any    `json:"payload"           bun:"payload,type:JSONB,nullzero"`
	MappingIDs        []string          `json:"mappingIds"        bun:"mapping_ids,type:TEXT[],array,notnull"`
	ErrorCategory     SyncErrorCategory `json:"errorCategory"     bun:"error_category,type:VARCHAR(30),nullzero"`
	ErrorCode         string            `json:"errorCode"         bun:"error_code,type:VARCHAR(50),nullzero"`
	ErrorMessage      string            `json:"errorMessage"      bun:"error_message,type:TEXT,nullzero"`
	Resolution        string            `json:"resolution"        bun:"resolution,type:TEXT,nullzero"`
	QueuedAt          int64             `json:"queuedAt"          bun:"queued_at,type:BIGINT,notnull"`
	StartedAt         *int64            `json:"startedAt"         bun:"started_at,type:BIGINT,nullzero"`
	SyncedAt          *int64            `json:"syncedAt"          bun:"synced_at,type:BIGINT,nullzero"`
	ReleasedByID      pulid.ID          `json:"releasedById"      bun:"released_by_id,type:VARCHAR(100),nullzero"`
	SkippedByID       pulid.ID          `json:"skippedById"       bun:"skipped_by_id,type:VARCHAR(100),nullzero"`
	SkippedReason     string            `json:"skippedReason"     bun:"skipped_reason,type:TEXT,nullzero"`
	Version           int64             `json:"version"           bun:"version,type:BIGINT"`
	CreatedAt         int64             `json:"createdAt"         bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64             `json:"updatedAt"         bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	SkippedBy    *tenant.User         `json:"skippedBy,omitempty"    bun:"rel:belongs-to,join:skipped_by_id=id"`
	ReleasedBy   *tenant.User         `json:"releasedBy,omitempty"   bun:"rel:belongs-to,join:released_by_id=id"`
}

type SyncRecordKey struct {
	ObjectType SyncObjectType
	ObjectID   pulid.ID
	Operation  SyncOperation
	Revision   int64
}

func (k SyncRecordKey) String() string {
	return string(k.ObjectType) + ":" + k.ObjectID.String() + ":" + string(k.Operation) + ":" +
		strconv.FormatInt(k.Revision, 10)
}

func SyncRequestID(connectionID pulid.ID, idempotencyKey string) string {
	sum := sha256.Sum256([]byte(connectionID.String() + "|" + idempotencyKey))
	return SyncRequestIDPrefix + hex.EncodeToString(sum[:])[:syncRequestIDHashChars]
}

func SyncStepRequestID(requestID string, step int) string {
	return requestID + "-" + strconv.Itoa(step)
}

type NewSyncRecord struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Key          SyncRecordKey
	ObjectNumber string
	SourceEvent  SyncSourceEvent
	DocumentDate *int64
	AwaitRelease bool
	DependsOn    pulid.ID
	At           int64
}

func NewAccountingSyncRecord(p *NewSyncRecord) *AccountingSyncRecord {
	status := SyncStatusQueued
	if p.AwaitRelease {
		status = SyncStatusAwaitingApproval
	}
	recordKey := p.Key
	recordKey.Revision = max(recordKey.Revision, 1)
	key := recordKey.String()
	at := p.At
	return &AccountingSyncRecord{
		ID:                pulid.MustNew(syncRecordIDPrefix),
		OrganizationID:    p.TenantInfo.OrgID,
		BusinessUnitID:    p.TenantInfo.BuID,
		ConnectionID:      p.ConnectionID,
		ObjectType:        recordKey.ObjectType,
		ObjectID:          recordKey.ObjectID,
		ObjectNumber:      stringutils.TruncateRunes(p.ObjectNumber, 100),
		Operation:         recordKey.Operation,
		SourceEvent:       p.SourceEvent,
		IdempotencyKey:    key,
		RequestID:         SyncRequestID(p.ConnectionID, key),
		Revision:          recordKey.Revision,
		DocumentDate:      p.DocumentDate,
		DependsOnRecordID: p.DependsOn,
		Status:            status,
		NextAttemptAt:     &at,
		ExternalRefs:      map[string]string{},
		MappingIDs:        []string{},
		QueuedAt:          p.At,
	}
}

func (r *AccountingSyncRecord) GetTableName() string { return "accounting_sync_records" }

func (r *AccountingSyncRecord) GetID() pulid.ID { return r.ID }

func (r *AccountingSyncRecord) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *AccountingSyncRecord) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *AccountingSyncRecord) GetCreatedAt() int64 { return r.CreatedAt }

func (r *AccountingSyncRecord) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "acctsr",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "object_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "external_doc_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "error_message",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (r *AccountingSyncRecord) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if r.ExternalRefs == nil {
		r.ExternalRefs = map[string]string{}
	}
	if r.MappingIDs == nil {
		r.MappingIDs = []string{}
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew(syncRecordIDPrefix)
		}
		if r.QueuedAt == 0 {
			r.QueuedAt = now
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}
	return nil
}

func (r *AccountingSyncRecord) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: r.OrganizationID, BuID: r.BusinessUnitID}
}

func (r *AccountingSyncRecord) StepRequestID(step int) string {
	return SyncStepRequestID(r.RequestID, step)
}

func (r *AccountingSyncRecord) SetExternalRef(key, value string) {
	if r.ExternalRefs == nil {
		r.ExternalRefs = map[string]string{}
	}
	if value == "" {
		delete(r.ExternalRefs, key)
		return
	}
	r.ExternalRefs[key] = value
}

func (r *AccountingSyncRecord) clearError() {
	r.ErrorCategory = ""
	r.ErrorCode = ""
	r.ErrorMessage = ""
	r.Resolution = ""
}

func (r *AccountingSyncRecord) setError(failure *SyncError) {
	r.ErrorCategory = failure.Category
	r.ErrorCode = stringutils.TruncateRunes(failure.Code, 50)
	r.ErrorMessage = stringutils.TruncateRunes(failure.Message, maxSyncErrorMessage)
	r.Resolution = stringutils.TruncateRunes(failure.Resolution, maxSyncResolution)
}

type SyncResult struct {
	ExternalID        string
	ExternalDocNumber string
	ExternalURL       string
	ExternalRefs      map[string]string
	PayloadHash       string
	Payload           map[string]any
	MappingIDs        []string
}

type SyncError struct {
	Category   SyncErrorCategory
	Code       string
	Message    string
	Resolution string
}

func (f *SyncError) Error() string { return f.Message }

func (r *AccountingSyncRecord) MarkSynced(result *SyncResult, at int64) {
	r.Status = SyncStatusSynced
	r.ExternalID = stringutils.TruncateRunes(result.ExternalID, 100)
	r.ExternalDocNumber = stringutils.TruncateRunes(result.ExternalDocNumber, 100)
	r.ExternalURL = result.ExternalURL
	for key, value := range result.ExternalRefs {
		r.SetExternalRef(key, value)
	}
	r.PayloadHash = result.PayloadHash
	r.Payload = result.Payload
	r.MappingIDs = append([]string{}, result.MappingIDs...)
	r.SyncedAt = &at
	r.NextAttemptAt = nil
	r.LeaseExpiresAt = nil
	r.clearError()
}

func SyncRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := SyncRetryBase
	for range attempt - 1 {
		delay *= 2
		if delay >= SyncRetryCeiling {
			return SyncRetryCeiling
		}
	}
	return delay
}

func (r *AccountingSyncRecord) MarkFailed(failure *SyncError, at int64) SyncAttemptOutcome {
	r.setError(failure)
	r.LeaseExpiresAt = nil
	switch {
	case failure.Category.WaitsOnConnection():
		next := at + int64(SyncAuthWait/time.Second)
		r.AttemptCount = max(r.AttemptCount-1, 0)
		r.Status = SyncStatusRetrying
		r.NextAttemptAt = &next
		return SyncAttemptWaiting
	case failure.Category.Retries() && r.AttemptCount < MaxSyncAttempts:
		next := at + int64(SyncRetryDelay(r.AttemptCount)/time.Second)
		r.Status = SyncStatusRetrying
		r.NextAttemptAt = &next
		return SyncAttemptRetrying
	case failure.Category.Retries():
		r.Status = SyncStatusDeadLettered
		r.NextAttemptAt = nil
		return SyncAttemptDeadLettered
	default:
		r.Status = SyncStatusBlocked
		r.NextAttemptAt = nil
		return SyncAttemptBlocked
	}
}

func (r *AccountingSyncRecord) Wait(failure *SyncError, at int64, delay time.Duration) {
	r.setError(failure)
	next := at + int64(delay/time.Second)
	r.AttemptCount = max(r.AttemptCount-1, 0)
	r.Status = SyncStatusQueued
	r.NextAttemptAt = &next
	r.LeaseExpiresAt = nil
}

func (r *AccountingSyncRecord) Retry(at int64) error {
	if !r.Status.Retryable() {
		return ErrSyncRecordNotRetryable
	}
	r.Status = SyncStatusQueued
	r.AttemptCount = 0
	r.NextAttemptAt = &at
	r.LeaseExpiresAt = nil
	return nil
}

func (r *AccountingSyncRecord) Release(actorID pulid.ID, at int64) error {
	if r.Status != SyncStatusAwaitingApproval {
		return ErrSyncRecordNotReleasable
	}
	r.Status = SyncStatusQueued
	r.ReleasedByID = actorID
	r.NextAttemptAt = &at
	return nil
}

func (r *AccountingSyncRecord) Skip(actorID pulid.ID, reason string) error {
	if r.Status.IsFinal() || r.Status == SyncStatusInFlight {
		return ErrSyncRecordNotSkippable
	}
	reason = stringutils.TruncateRunes(
		stringutils.OneLine(reason, maxSyncSkipReason),
		maxSyncSkipReason,
	)
	if reason == "" {
		return ErrSkipReasonRequired
	}
	r.Status = SyncStatusSkipped
	r.SkippedByID = actorID
	r.SkippedReason = reason
	r.NextAttemptAt = nil
	r.LeaseExpiresAt = nil
	return nil
}

func (r *AccountingSyncRecord) Supersede() bool {
	if r.Status.IsFinal() || r.Status == SyncStatusInFlight {
		return false
	}
	r.Status = SyncStatusSuperseded
	r.NextAttemptAt = nil
	r.LeaseExpiresAt = nil
	return true
}

func (r *AccountingSyncRecord) NeverSent() bool {
	return r.Status != SyncStatusSynced && r.ExternalID == "" && len(r.ExternalRefs) == 0
}

type AccountingSyncAttempt struct {
	bun.BaseModel `bun:"table:accounting_sync_attempts,alias:acctsa" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SyncRecordID   pulid.ID           `json:"syncRecordId"   bun:"sync_record_id,type:VARCHAR(100),notnull"`
	AttemptNumber  int                `json:"attemptNumber"  bun:"attempt_number,type:INTEGER,notnull"`
	Outcome        SyncAttemptOutcome `json:"outcome"        bun:"outcome,type:VARCHAR(20),notnull"`
	ErrorCategory  SyncErrorCategory  `json:"errorCategory"  bun:"error_category,type:VARCHAR(30),nullzero"`
	ErrorCode      string             `json:"errorCode"      bun:"error_code,type:VARCHAR(50),nullzero"`
	ErrorMessage   string             `json:"errorMessage"   bun:"error_message,type:TEXT,nullzero"`
	StartedAt      int64              `json:"startedAt"      bun:"started_at,type:BIGINT,notnull"`
	FinishedAt     int64              `json:"finishedAt"     bun:"finished_at,type:BIGINT,notnull"`
	DurationMs     int                `json:"durationMs"     bun:"duration_ms,type:INTEGER,notnull"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (a *AccountingSyncAttempt) GetTableName() string { return "accounting_sync_attempts" }

func (a *AccountingSyncAttempt) GetID() pulid.ID { return a.ID }

func (a *AccountingSyncAttempt) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *AccountingSyncAttempt) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *AccountingSyncAttempt) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if a.ID.IsNil() {
			a.ID = pulid.MustNew(syncAttemptIDPrefix)
		}
		a.CreatedAt = timeutils.NowUnix()
	}
	return nil
}

type NewSyncAttemptParams struct {
	Record        *AccountingSyncRecord
	AttemptNumber int
	Outcome       SyncAttemptOutcome
	StartedAt     time.Time
	FinishedAt    time.Time
}

func NewSyncAttempt(p *NewSyncAttemptParams) *AccountingSyncAttempt {
	attempt := &AccountingSyncAttempt{
		ID:             pulid.MustNew(syncAttemptIDPrefix),
		OrganizationID: p.Record.OrganizationID,
		BusinessUnitID: p.Record.BusinessUnitID,
		SyncRecordID:   p.Record.ID,
		AttemptNumber:  max(p.AttemptNumber, 1),
		Outcome:        p.Outcome,
		StartedAt:      p.StartedAt.Unix(),
		FinishedAt:     p.FinishedAt.Unix(),
		DurationMs:     int(max(p.FinishedAt.Sub(p.StartedAt), 0) / time.Millisecond),
	}
	if p.Outcome != SyncAttemptSynced {
		attempt.ErrorCategory = p.Record.ErrorCategory
		attempt.ErrorCode = p.Record.ErrorCode
		attempt.ErrorMessage = p.Record.ErrorMessage
	}
	return attempt
}

type BackfillCursor struct {
	ObjectType SyncObjectType `json:"objectType"`
	AfterAt    int64          `json:"afterAt"`
	AfterID    pulid.ID       `json:"afterId"`
}

type AccountingBackfill struct {
	bun.BaseModel `bun:"table:accounting_backfills,alias:acctbf" json:"-"`

	ID                 pulid.ID         `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID     pulid.ID         `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID         `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ConnectionID       pulid.ID         `json:"connectionId"       bun:"connection_id,type:VARCHAR(100),notnull"`
	RangeStart         int64            `json:"rangeStart"         bun:"range_start,type:BIGINT,notnull"`
	RangeEnd           int64            `json:"rangeEnd"           bun:"range_end,type:BIGINT,notnull"`
	ObjectTypes        []SyncObjectType `json:"objectTypes"        bun:"object_types,type:TEXT[],array,notnull"`
	Cursor             BackfillCursor   `json:"cursor"             bun:"cursor,type:JSONB,notnull"`
	Status             BackfillStatus   `json:"status"             bun:"status,type:VARCHAR(20),notnull"`
	EnqueuedCount      int              `json:"enqueuedCount"      bun:"enqueued_count,type:INTEGER,notnull"`
	AlreadyQueuedCount int              `json:"alreadyQueuedCount" bun:"already_queued_count,type:INTEGER,notnull"`
	RequestedByID      pulid.ID         `json:"requestedById"      bun:"requested_by_id,type:VARCHAR(100),nullzero"`
	StartedAt          *int64           `json:"startedAt"          bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt        *int64           `json:"completedAt"        bun:"completed_at,type:BIGINT,nullzero"`
	LastError          string           `json:"lastError"          bun:"last_error,type:TEXT,nullzero"`
	Version            int64            `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64            `json:"createdAt"          bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64            `json:"updatedAt"          bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (b *AccountingBackfill) GetTableName() string { return "accounting_backfills" }

func (b *AccountingBackfill) GetID() pulid.ID { return b.ID }

func (b *AccountingBackfill) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *AccountingBackfill) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *AccountingBackfill) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew(backfillIDPrefix)
		}
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}
	return nil
}

func (b *AccountingBackfill) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: b.OrganizationID, BuID: b.BusinessUnitID}
}

type NewBackfillParams struct {
	TenantInfo    pagination.TenantInfo
	ConnectionID  pulid.ID
	RangeStart    int64
	RangeEnd      int64
	ObjectTypes   []SyncObjectType
	RequestedByID pulid.ID
}

func NewAccountingBackfill(p *NewBackfillParams) *AccountingBackfill {
	types := p.ObjectTypes
	if len(types) == 0 {
		types = BackfillObjectTypes()
	}
	return &AccountingBackfill{
		ID:             pulid.MustNew(backfillIDPrefix),
		OrganizationID: p.TenantInfo.OrgID,
		BusinessUnitID: p.TenantInfo.BuID,
		ConnectionID:   p.ConnectionID,
		RangeStart:     p.RangeStart,
		RangeEnd:       max(p.RangeEnd, p.RangeStart),
		ObjectTypes:    types,
		Cursor:         BackfillCursor{ObjectType: types[0]},
		Status:         BackfillStatusQueued,
		RequestedByID:  p.RequestedByID,
	}
}

func BackfillObjectTypes() []SyncObjectType {
	return []SyncObjectType{
		SyncObjectInvoice,
		SyncObjectDebitMemo,
		SyncObjectCreditMemo,
		SyncObjectCustomerPayment,
		SyncObjectCreditApplication,
		SyncObjectCarrierBill,
		SyncObjectCarrierBillPay,
		SyncObjectDriverBill,
		SyncObjectDriverBillPay,
	}
}

func (b *AccountingBackfill) Start(at int64) bool {
	if b.Status != BackfillStatusQueued && b.Status != BackfillStatusRunning {
		return false
	}
	b.Status = BackfillStatusRunning
	if b.StartedAt == nil {
		b.StartedAt = &at
	}
	return true
}

func (b *AccountingBackfill) Advance(cursor BackfillCursor, enqueued, existing int) {
	b.Cursor = cursor
	b.EnqueuedCount += enqueued
	b.AlreadyQueuedCount += existing
}

func (b *AccountingBackfill) NextObjectType() bool {
	idx := slices.Index(b.ObjectTypes, b.Cursor.ObjectType)
	if idx < 0 || idx+1 >= len(b.ObjectTypes) {
		return false
	}
	b.Cursor = BackfillCursor{ObjectType: b.ObjectTypes[idx+1]}
	return true
}

func (b *AccountingBackfill) Pause() bool {
	if b.Status != BackfillStatusQueued && b.Status != BackfillStatusRunning {
		return false
	}
	b.Status = BackfillStatusPaused
	return true
}

func (b *AccountingBackfill) Resume() bool {
	if b.Status != BackfillStatusPaused {
		return false
	}
	b.Status = BackfillStatusQueued
	return true
}

func (b *AccountingBackfill) Cancel(at int64) bool {
	if !b.Status.InProgress() {
		return false
	}
	b.Status = BackfillStatusCancelled
	b.CompletedAt = &at
	return true
}

func (b *AccountingBackfill) Complete(at int64) {
	b.Status = BackfillStatusCompleted
	b.CompletedAt = &at
	b.LastError = ""
}

func (b *AccountingBackfill) Fail(message string, at int64) {
	b.Status = BackfillStatusFailed
	b.CompletedAt = &at
	b.LastError = stringutils.TruncateRunes(message, maxSyncErrorMessage)
}
