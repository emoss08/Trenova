package accountingsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
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
	ExternalRefCreditDocument = "creditDocument"
	ExternalRefAdopted        = "adopted"
	ExternalRefURL            = "url"
	ExternalRefCombined       = "combined"
	ExternalRefReflectedIn    = "reflectedIn"
)

var (
	ErrSyncRecordNotRetryable = errors.New(
		"only a blocked, dead-lettered or retrying record can be retried",
	)
	ErrSyncRecordNotSkippable  = errors.New("a record already sent or skipped cannot be skipped")
	ErrSyncRecordNotReleasable = errors.New("only a record waiting for approval can be released")
	ErrSkipReasonRequired      = errors.New("a reason is required to skip a record")
	ErrSyncRecordNotRedatable  = errors.New(
		"only a record blocked by a closed period can be sent on the first open day",
	)
)

// Every document sent to the accounting system and what happened to it
type AccountingSyncRecord struct {
	keel.Table `keel:"accounting_sync_records,report=Accounting"`

	ID                pulid.ID          `json:"id"                keel:"id,pk,autoid:acctsr_"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID          `json:"organizationId"    keel:"organization_id,pk,tenant"`
	ConnectionID      pulid.ID          `json:"connectionId"      keel:"connection_id"`
	ObjectType        SyncObjectType    `json:"objectType"        keel:"object_type,label=Document Type"`
	ObjectID          pulid.ID          `json:"objectId"          keel:"object_id,label=Document ID"`
	ObjectNumber      string            `json:"objectNumber"      keel:"object_number,nullzero,label=Document Number"`
	Operation         SyncOperation     `json:"operation"         keel:"operation,label=Action"`
	SourceEvent       SyncSourceEvent   `json:"sourceEvent"       keel:"source_event,label=Queued By"`
	IdempotencyKey    string            `json:"idempotencyKey"    keel:"idempotency_key,noreport"`
	RequestID         string            `json:"requestId"         keel:"request_id,noreport"`
	Revision          int64             `json:"revision"          keel:"revision,dimension"`
	DocumentDate      *int64            `json:"documentDate"      keel:"document_date"`
	DependsOnRecordID pulid.ID          `json:"dependsOnRecordId" keel:"depends_on_record_id,nullzero,noreport"`
	Status            SyncStatus        `json:"status"            keel:"status"`
	AttemptCount      int               `json:"attemptCount"      keel:"attempt_count,label=Tries,format=count,dimension"`
	NextAttemptAt     *int64            `json:"nextAttemptAt"     keel:"next_attempt_at,label=Next Try"`
	LeaseExpiresAt    *int64            `json:"-"                 keel:"lease_expires_at,noreport"`
	ExternalID        string            `json:"externalId"        keel:"external_id,nullzero,label=ID in the Books"`
	ExternalDocNumber string            `json:"externalDocNumber" keel:"external_doc_number,nullzero,label=Number in the Books"`
	ExternalURL       string            `json:"externalUrl"       keel:"external_url,nullzero,label=Link in the Books"`
	ExternalRefs      map[string]string `json:"externalRefs"      keel:"external_refs,jsonb,noreport"`
	PayloadHash       string            `json:"payloadHash"       keel:"payload_hash,nullzero,noreport"`
	Payload           map[string]any    `json:"payload"           keel:"payload,jsonb,nullzero,noreport"`
	MappingIDs        []string          `json:"mappingIds"        keel:"mapping_ids,noreport"`
	ErrorCategory     SyncErrorCategory `json:"errorCategory"     keel:"error_category,nullzero"`
	ErrorCode         string            `json:"errorCode"         keel:"error_code,nullzero"`
	ErrorMessage      string            `json:"errorMessage"      keel:"error_message,nullzero"`
	Resolution        string            `json:"resolution"        keel:"resolution,nullzero,label=What to Do"`
	QueuedAt          int64             `json:"queuedAt"          keel:"queued_at"`
	StartedAt         *int64            `json:"startedAt"         keel:"started_at"`
	SyncedAt          *int64            `json:"syncedAt"          keel:"synced_at"`
	ReleasedByID      pulid.ID          `json:"releasedById"      keel:"released_by_id,nullzero"`
	SkippedByID       pulid.ID          `json:"skippedById"       keel:"skipped_by_id,nullzero"`
	SkippedReason     string            `json:"skippedReason"     keel:"skipped_reason,nullzero,label=Skip Reason"`
	RedatedTo         *int64            `json:"redatedTo"         keel:"redated_to,type=epoch"`
	RedatedByID       pulid.ID          `json:"redatedById"       keel:"redated_by_id,nullzero"`
	Version           int64             `json:"version"           keel:"version,version"`
	CreatedAt         int64             `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64             `json:"updatedAt"         keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	SkippedBy    *tenant.User         `json:"skippedBy,omitempty"    keel:"belongs:skipped_by_id,noreport"`
	ReleasedBy   *tenant.User         `json:"releasedBy,omitempty"   keel:"belongs:released_by_id,noreport"`
}

func (r *AccountingSyncRecord) BeforeUpdate(_ context.Context) error {
	r.fillCollections()

	return nil
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

func (r *AccountingSyncRecord) BeforeCreate(_ context.Context) error {
	r.fillCollections()
	if r.QueuedAt == 0 {
		r.QueuedAt = timeutils.NowUnix()
	}

	return nil
}

func (r *AccountingSyncRecord) BeforeSave(_ context.Context) error {
	r.fillCollections()
	return nil
}

func (r *AccountingSyncRecord) fillCollections() {
	if r.ExternalRefs == nil {
		r.ExternalRefs = map[string]string{}
	}
	if r.MappingIDs == nil {
		r.MappingIDs = []string{}
	}
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
	RetryAfter time.Duration
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

type SyncLink struct {
	ExternalID  string
	ExternalURL string
	Resolution  string
	Combined    bool
}

func (r *AccountingSyncRecord) Link(link *SyncLink, at int64) bool {
	if r.Status.IsFinal() || r.Status == SyncStatusInFlight || link.ExternalID == "" {
		return false
	}
	r.Status = SyncStatusSynced
	r.ExternalID = stringutils.TruncateRunes(link.ExternalID, 100)
	r.ExternalURL = link.ExternalURL
	if link.Combined {
		r.SetExternalRef(ExternalRefCombined, link.ExternalID)
	}
	r.SyncedAt = &at
	r.NextAttemptAt = nil
	r.LeaseExpiresAt = nil
	r.clearError()
	r.Resolution = stringutils.TruncateRunes(link.Resolution, maxSyncResolution)
	return true
}

func (r *AccountingSyncRecord) SharesProviderDocument() bool {
	return r.ExternalRefs[ExternalRefCombined] != ""
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
		delay := max(SyncRetryDelay(r.AttemptCount), min(failure.RetryAfter, SyncRetryCeiling))
		next := at + int64(delay/time.Second)
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

func (r *AccountingSyncRecord) Redate(actorID pulid.ID, sentDate, at int64) error {
	if r.Status != SyncStatusBlocked || r.ErrorCategory != SyncErrorClosedPeriod {
		return ErrSyncRecordNotRedatable
	}
	r.RedatedTo = &sentDate
	r.RedatedByID = actorID
	r.Status = SyncStatusQueued
	r.AttemptCount = 0
	r.NextAttemptAt = &at
	r.LeaseExpiresAt = nil
	return nil
}

func (r *AccountingSyncRecord) SentDate(documentDate int64) int64 {
	if r.RedatedTo != nil {
		return *r.RedatedTo
	}
	return documentDate
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

func (r *AccountingSyncRecord) Reflect(actorID pulid.ID, externalID, reason string) bool {
	if r.Status.IsFinal() || r.Status == SyncStatusInFlight || externalID == "" {
		return false
	}
	r.Status = SyncStatusSkipped
	r.SkippedByID = actorID
	r.SkippedReason = stringutils.TruncateRunes(
		stringutils.OneLine(reason, maxSyncSkipReason),
		maxSyncSkipReason,
	)
	r.SetExternalRef(ExternalRefReflectedIn, externalID)
	r.NextAttemptAt = nil
	r.LeaseExpiresAt = nil
	r.clearError()
	r.Resolution = stringutils.TruncateRunes(reason, maxSyncResolution)
	return true
}

func (r *AccountingSyncRecord) ReflectedIn() string {
	if r.Status != SyncStatusSkipped {
		return ""
	}
	return r.ExternalRefs[ExternalRefReflectedIn]
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
	keel.Table `keel:"accounting_sync_attempts"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:acctsa_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	SyncRecordID   pulid.ID           `json:"syncRecordId"   keel:"sync_record_id"`
	AttemptNumber  int                `json:"attemptNumber"  keel:"attempt_number"`
	Outcome        SyncAttemptOutcome `json:"outcome"        keel:"outcome"`
	ErrorCategory  SyncErrorCategory  `json:"errorCategory"  keel:"error_category,nullzero"`
	ErrorCode      string             `json:"errorCode"      keel:"error_code,nullzero"`
	ErrorMessage   string             `json:"errorMessage"   keel:"error_message,nullzero"`
	StartedAt      int64              `json:"startedAt"      keel:"started_at"`
	FinishedAt     int64              `json:"finishedAt"     keel:"finished_at"`
	DurationMs     int                `json:"durationMs"     keel:"duration_ms"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
}

func (a *AccountingSyncAttempt) GetTableName() string { return "accounting_sync_attempts" }

func (a *AccountingSyncAttempt) GetID() pulid.ID { return a.ID }

func (a *AccountingSyncAttempt) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *AccountingSyncAttempt) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

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
	keel.Table `keel:"accounting_backfills"`

	ID                 pulid.ID         `json:"id"                 keel:"id,pk,autoid:acctbf_"`
	BusinessUnitID     pulid.ID         `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID         `json:"organizationId"     keel:"organization_id,pk,tenant"`
	ConnectionID       pulid.ID         `json:"connectionId"       keel:"connection_id"`
	RangeStart         int64            `json:"rangeStart"         keel:"range_start"`
	RangeEnd           int64            `json:"rangeEnd"           keel:"range_end"`
	ObjectTypes        []SyncObjectType `json:"objectTypes"        keel:"object_types"`
	Cursor             BackfillCursor   `json:"cursor"             keel:"cursor,jsonb"`
	Status             BackfillStatus   `json:"status"             keel:"status"`
	EnqueuedCount      int              `json:"enqueuedCount"      keel:"enqueued_count"`
	AlreadyQueuedCount int              `json:"alreadyQueuedCount" keel:"already_queued_count"`
	RequestedByID      pulid.ID         `json:"requestedById"      keel:"requested_by_id,nullzero"`
	StartedAt          *int64           `json:"startedAt"          keel:"started_at"`
	CompletedAt        *int64           `json:"completedAt"        keel:"completed_at"`
	LastError          string           `json:"lastError"          keel:"last_error,nullzero"`
	Version            int64            `json:"version"            keel:"version,version"`
	CreatedAt          int64            `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64            `json:"updatedAt"          keel:"updated_at,autoupdate"`
}

func (b *AccountingBackfill) GetTableName() string { return "accounting_backfills" }

func (b *AccountingBackfill) GetID() pulid.ID { return b.ID }

func (b *AccountingBackfill) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *AccountingBackfill) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

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
		SyncObjectJournalEntry,
		SyncObjectJournalSummary,
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

func IsCreditDocument(refs map[string]string) bool {
	return refs[ExternalRefCreditDocument] == "true" ||
		refs[ExternalRefDocumentType] == "VendorCredit"
}
