package airetrieval

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*IndexEntry)(nil)

type IndexEntry struct {
	bun.BaseModel `bun:"table:ai_index_entries,alias:aie" json:"-"`
	keel.Table    `keel:"ai_index_entries"`

	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	SourceType     SourceType  `json:"sourceType"     bun:"source_type,pk,type:VARCHAR(30),notnull" keel:"source_type,pk"`
	SourceID       pulid.ID    `json:"sourceId"       bun:"source_id,pk,type:VARCHAR(100),notnull" keel:"source_id,pk"`
	ModelKey       string      `json:"modelKey"       bun:"model_key,pk,type:VARCHAR(300),notnull" keel:"model_key,pk"`
	Status         IndexStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull,default:'Pending'" keel:"status"`
	Generation     int64       `json:"generation"     bun:"generation,type:BIGINT,notnull,default:1" keel:"generation"`
	Attempts       int         `json:"attempts"       bun:"attempts,type:INTEGER,notnull,default:0" keel:"attempts"`
	ChunkCount     int         `json:"chunkCount"     bun:"chunk_count,type:INTEGER,notnull,default:0" keel:"chunk_count"`
	LastError      string      `json:"lastError"      bun:"last_error,type:TEXT,nullzero" keel:"last_error,nullzero"`
	LeaseExpiresAt *int64      `json:"leaseExpiresAt" bun:"lease_expires_at,type:BIGINT,nullzero" keel:"lease_expires_at"`
	NextAttemptAt  *int64      `json:"nextAttemptAt"  bun:"next_attempt_at,type:BIGINT,nullzero" keel:"next_attempt_at"`
	LastAttemptAt  *int64      `json:"lastAttemptAt"  bun:"last_attempt_at,type:BIGINT,nullzero" keel:"last_attempt_at"`
	IndexedAt      *int64      `json:"indexedAt"      bun:"indexed_at,type:BIGINT,nullzero" keel:"indexed_at"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64       `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (e *IndexEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.CreatedAt == 0 {
			e.CreatedAt = now
		}
		if e.UpdatedAt == 0 {
			e.UpdatedAt = now
		}
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
