package aiaudit

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// AIAuditChainHead is where a tenant's chain ends and what its last check
// found. The projector takes a row lock on it to assign the next seq, which
// is what keeps the chain single-file.
type AIAuditChainHead struct {
	keel.Table `keel:"ai_audit_chain_heads"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`

	LastSeq   int64  `json:"lastSeq" keel:"last_seq"`
	LastHash  string `json:"lastHash" keel:"last_hash"`
	HashKeyID string `json:"hashKeyId" keel:"hash_key_id,nullzero"`

	LastVerifiedSeq           int64              `json:"lastVerifiedSeq" keel:"last_verified_seq"`
	LastVerifiedAt            *int64             `json:"lastVerifiedAt" keel:"last_verified_at"`
	LastVerificationStatus    VerificationStatus `json:"lastVerificationStatus" keel:"last_verification_status,nullzero"`
	LastVerificationFailedSeq *int64             `json:"lastVerificationFailedSeq" keel:"last_verification_failed_seq"`
	LastVerificationDetail    string             `json:"lastVerificationDetail" keel:"last_verification_detail,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (h *AIAuditChainHead) GetTableName() string {
	return "ai_audit_chain_heads"
}

// AIAuditSeal checkpoints one projector batch of a tenant's chain: the range
// it covered and the hash it ended on. Seals outlive the rows they cover, so a
// chain pruned for retention still verifies from the seal before its oldest
// row.
type AIAuditSeal struct {
	keel.Table `keel:"ai_audit_seals"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ToSeq          int64    `json:"toSeq" keel:"to_seq,pk"`

	FromSeq   int64  `json:"fromSeq" keel:"from_seq"`
	HeadHash  string `json:"headHash" keel:"head_hash"`
	HashKeyID string `json:"hashKeyId" keel:"hash_key_id,nullzero"`
	RowCount  int    `json:"rowCount" keel:"row_count"`
	SealedAt  int64  `json:"sealedAt" keel:"sealed_at"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (s *AIAuditSeal) GetTableName() string {
	return "ai_audit_seals"
}

// AIAuditProjectorState is how far the projector has read one source. The
// watermark is the source row it last read, by the timestamp column the
// source is scanned by and its id.
type AIAuditProjectorState struct {
	keel.Table `keel:"ai_audit_projector_state"`

	Source      Source `json:"source" keel:"source,pk"`
	WatermarkTS int64  `json:"watermarkTs" keel:"watermark_ts"`
	WatermarkID string `json:"watermarkId" keel:"watermark_id"`
	UpdatedAt   int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (s *AIAuditProjectorState) GetTableName() string {
	return "ai_audit_projector_state"
}

func (s *AIAuditSeal) BeforeCreate(_ context.Context) error {
	if s.SealedAt == 0 {
		s.SealedAt = timeutils.NowUnix()
	}

	return nil
}
