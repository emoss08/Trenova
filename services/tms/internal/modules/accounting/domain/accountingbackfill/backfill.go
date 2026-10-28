package accountingbackfill

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	backfillIDPrefix = "acctbf_"
	maxBackfillError = 2000
)

type BackfillStatus string

const (
	BackfillStatusQueued    = BackfillStatus("Queued")
	BackfillStatusRunning   = BackfillStatus("Running")
	BackfillStatusPaused    = BackfillStatus("Paused")
	BackfillStatusCompleted = BackfillStatus("Completed")
	BackfillStatusFailed    = BackfillStatus("Failed")
	BackfillStatusCancelled = BackfillStatus("Cancelled")
)

func (s BackfillStatus) String() string { return string(s) }

func (s BackfillStatus) IsValid() bool {
	switch s {
	case BackfillStatusQueued,
		BackfillStatusRunning,
		BackfillStatusPaused,
		BackfillStatusCompleted,
		BackfillStatusFailed,
		BackfillStatusCancelled:
		return true
	default:
		return false
	}
}

func (s BackfillStatus) InProgress() bool {
	return s == BackfillStatusQueued || s == BackfillStatusRunning || s == BackfillStatusPaused
}

func AllBackfillStatuses() []BackfillStatus {
	return []BackfillStatus{
		BackfillStatusQueued,
		BackfillStatusRunning,
		BackfillStatusPaused,
		BackfillStatusCompleted,
		BackfillStatusFailed,
		BackfillStatusCancelled,
	}
}

type BackfillCursor struct {
	ObjectType accountingsync.SyncObjectType `json:"objectType"`
	AfterAt    int64                         `json:"afterAt"`
	AfterID    pulid.ID                      `json:"afterId"`
}

type AccountingBackfill struct {
	keel.Table `keel:"accounting_backfills"`

	ID                 pulid.ID                        `json:"id"                 keel:"id,pk,autoid:acctbf_"`
	BusinessUnitID     pulid.ID                        `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID                        `json:"organizationId"     keel:"organization_id,pk,tenant"`
	ConnectionID       pulid.ID                        `json:"connectionId"       keel:"connection_id"`
	RangeStart         int64                           `json:"rangeStart"         keel:"range_start"`
	RangeEnd           int64                           `json:"rangeEnd"           keel:"range_end"`
	ObjectTypes        []accountingsync.SyncObjectType `json:"objectTypes"        keel:"object_types"`
	Cursor             BackfillCursor                  `json:"cursor"             keel:"cursor,jsonb"`
	Status             BackfillStatus                  `json:"status"             keel:"status"`
	EnqueuedCount      int                             `json:"enqueuedCount"      keel:"enqueued_count"`
	AlreadyQueuedCount int                             `json:"alreadyQueuedCount" keel:"already_queued_count"`
	RequestedByID      pulid.ID                        `json:"requestedById"      keel:"requested_by_id,nullzero"`
	StartedAt          *int64                          `json:"startedAt"          keel:"started_at"`
	CompletedAt        *int64                          `json:"completedAt"        keel:"completed_at"`
	LastError          string                          `json:"lastError"          keel:"last_error,nullzero"`
	Version            int64                           `json:"version"            keel:"version,version"`
	CreatedAt          int64                           `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64                           `json:"updatedAt"          keel:"updated_at,autoupdate"`
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
	ObjectTypes   []accountingsync.SyncObjectType
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

func BackfillObjectTypes() []accountingsync.SyncObjectType {
	return []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectDebitMemo,
		accountingsync.SyncObjectCreditMemo,
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectCarrierBillPay,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay,
		accountingsync.SyncObjectJournalEntry,
		accountingsync.SyncObjectJournalSummary,
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
	b.LastError = stringutils.TruncateRunes(message, maxBackfillError)
}
