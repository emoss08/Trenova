package aiaudit

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*AIAuditChainHead)(nil)
	_ bun.BeforeAppendModelHook = (*AIAuditSeal)(nil)
	_ bun.BeforeAppendModelHook = (*AIAuditProjectorState)(nil)
)

// AIAuditChainHead is where a tenant's chain ends and what its last check
// found. The projector takes a row lock on it to assign the next seq, which
// is what keeps the chain single-file.
type AIAuditChainHead struct {
	bun.BaseModel `bun:"table:ai_audit_chain_heads,alias:aiach" json:"-"`
	keel.Table    `keel:"ai_audit_chain_heads"`

	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" keel:"business_unit_id,pk,tenant"`

	LastSeq   int64  `json:"lastSeq"   bun:"last_seq,type:BIGINT,notnull,default:0" keel:"last_seq"`
	LastHash  string `json:"lastHash"  bun:"last_hash,type:VARCHAR(64),notnull,default:''" keel:"last_hash"`
	HashKeyID string `json:"hashKeyId" bun:"hash_key_id,type:VARCHAR(40),nullzero" keel:"hash_key_id,nullzero"`

	LastVerifiedSeq           int64              `json:"lastVerifiedSeq"           bun:"last_verified_seq,type:BIGINT,notnull,default:0" keel:"last_verified_seq"`
	LastVerifiedAt            *int64             `json:"lastVerifiedAt"            bun:"last_verified_at,type:BIGINT,nullzero" keel:"last_verified_at"`
	LastVerificationStatus    VerificationStatus `json:"lastVerificationStatus"    bun:"last_verification_status,type:VARCHAR(20),nullzero" keel:"last_verification_status,nullzero"`
	LastVerificationFailedSeq *int64             `json:"lastVerificationFailedSeq" bun:"last_verification_failed_seq,type:BIGINT,nullzero" keel:"last_verification_failed_seq"`
	LastVerificationDetail    string             `json:"lastVerificationDetail"    bun:"last_verification_detail,type:TEXT,nullzero" keel:"last_verification_detail,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (h *AIAuditChainHead) GetTableName() string {
	return "ai_audit_chain_heads"
}

func (h *AIAuditChainHead) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		h.CreatedAt = now
		h.UpdatedAt = now
	case *bun.UpdateQuery:
		h.UpdatedAt = now
	}

	return nil
}

// AIAuditSeal checkpoints one projector batch of a tenant's chain: the range
// it covered and the hash it ended on. Seals outlive the rows they cover, so a
// chain pruned for retention still verifies from the seal before its oldest
// row.
type AIAuditSeal struct {
	bun.BaseModel `bun:"table:ai_audit_seals,alias:aias" json:"-"`
	keel.Table    `keel:"ai_audit_seals"`

	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" keel:"business_unit_id,pk,tenant"`
	ToSeq          int64    `json:"toSeq"          bun:"to_seq,pk,type:BIGINT,notnull" keel:"to_seq,pk"`

	FromSeq   int64  `json:"fromSeq"   bun:"from_seq,type:BIGINT,notnull" keel:"from_seq"`
	HeadHash  string `json:"headHash"  bun:"head_hash,type:VARCHAR(64),notnull" keel:"head_hash"`
	HashKeyID string `json:"hashKeyId" bun:"hash_key_id,type:VARCHAR(40),nullzero" keel:"hash_key_id,nullzero"`
	RowCount  int    `json:"rowCount"  bun:"row_count,type:INTEGER,notnull" keel:"row_count"`
	SealedAt  int64  `json:"sealedAt"  bun:"sealed_at,type:BIGINT,notnull" keel:"sealed_at"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (s *AIAuditSeal) GetTableName() string {
	return "ai_audit_seals"
}

func (s *AIAuditSeal) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok && s.SealedAt == 0 {
		s.SealedAt = timeutils.NowUnix()
	}

	return nil
}

// AIAuditProjectorState is how far the projector has read one source. The
// watermark is the source row it last read, by the timestamp column the
// source is scanned by and its id.
type AIAuditProjectorState struct {
	bun.BaseModel `bun:"table:ai_audit_projector_state,alias:aiaps" json:"-"`
	keel.Table    `keel:"ai_audit_projector_state"`

	Source      Source `json:"source"      bun:"source,pk,type:VARCHAR(40),notnull" keel:"source,pk"`
	WatermarkTS int64  `json:"watermarkTs" bun:"watermark_ts,type:BIGINT,notnull,default:0" keel:"watermark_ts"`
	WatermarkID string `json:"watermarkId" bun:"watermark_id,type:VARCHAR(100),notnull,default:''" keel:"watermark_id"`
	UpdatedAt   int64  `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (s *AIAuditProjectorState) GetTableName() string {
	return "ai_audit_projector_state"
}

func (s *AIAuditProjectorState) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery, *bun.UpdateQuery:
		s.UpdatedAt = timeutils.NowUnix()
	}

	return nil
}

func (s *AIAuditSeal) BeforeCreate(_ context.Context) error {
	if s.SealedAt == 0 {
		s.SealedAt = timeutils.NowUnix()
	}

	return nil
}
