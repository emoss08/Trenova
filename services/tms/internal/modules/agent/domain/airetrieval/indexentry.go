package airetrieval

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type IndexEntry struct {
	keel.Table `keel:"ai_index_entries"`

	OrganizationID pulid.ID    `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	SourceType     SourceType  `json:"sourceType"     keel:"source_type,pk"`
	SourceID       pulid.ID    `json:"sourceId"       keel:"source_id,pk"`
	ModelKey       string      `json:"modelKey"       keel:"model_key,pk"`
	Status         IndexStatus `json:"status"         keel:"status"`
	Generation     int64       `json:"generation"     keel:"generation"`
	Attempts       int         `json:"attempts"       keel:"attempts"`
	ChunkCount     int         `json:"chunkCount"     keel:"chunk_count"`
	LastError      string      `json:"lastError"      keel:"last_error,nullzero"`
	LeaseExpiresAt *int64      `json:"leaseExpiresAt" keel:"lease_expires_at"`
	NextAttemptAt  *int64      `json:"nextAttemptAt"  keel:"next_attempt_at"`
	LastAttemptAt  *int64      `json:"lastAttemptAt"  keel:"last_attempt_at"`
	IndexedAt      *int64      `json:"indexedAt"      keel:"indexed_at"`
	CreatedAt      int64       `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64       `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (e *IndexEntry) BeforeCreate(_ context.Context) error {
	if e.Status == "" {
		e.Status = IndexStatusPending
	}
	if e.Generation == 0 {
		e.Generation = 1
	}

	return nil
}

type IndexEntryKey struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	SourceType     SourceType
	SourceID       pulid.ID
	ModelKey       string
}

func (k *IndexEntryKey) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: k.OrganizationID, BuID: k.BusinessUnitID}
}

func (e *IndexEntry) Key() IndexEntryKey {
	return IndexEntryKey{
		OrganizationID: e.OrganizationID,
		BusinessUnitID: e.BusinessUnitID,
		SourceType:     e.SourceType,
		SourceID:       e.SourceID,
		ModelKey:       e.ModelKey,
	}
}

func (e *IndexEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *IndexEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *IndexEntry) GetTableName() string { return "ai_index_entries" }
