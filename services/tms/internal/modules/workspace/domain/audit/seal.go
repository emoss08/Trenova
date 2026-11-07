package audit

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type VerificationStatus string

const (
	VerificationVerified   = VerificationStatus("Verified")
	VerificationBroken     = VerificationStatus("Broken")
	VerificationKeyMissing = VerificationStatus("KeyMissing")
)

const (
	SealHashVersionSigned   int16 = 1
	SealHashVersionUnsigned int16 = 2
	SealWindowSeconds             = int64(3600)
)

type ChainHead struct {
	keel.Table `keel:"audit_chain_heads"`

	OrganizationID            pulid.ID           `json:"organizationId"            keel:"organization_id,pk,tenant"`
	BusinessUnitID            pulid.ID           `json:"businessUnitId"            keel:"business_unit_id,pk,tenant"`
	LastSeq                   int64              `json:"lastSeq"                   keel:"last_seq"`
	LastHash                  string             `json:"lastHash"                  keel:"last_hash"`
	SealedThrough             int64              `json:"sealedThrough"             keel:"sealed_through"`
	HashKeyID                 string             `json:"hashKeyId"                 keel:"hash_key_id,nullzero"`
	LastVerifiedSeq           int64              `json:"lastVerifiedSeq"           keel:"last_verified_seq"`
	LastVerifiedAt            *int64             `json:"lastVerifiedAt"            keel:"last_verified_at"`
	LastVerificationStatus    VerificationStatus `json:"lastVerificationStatus"    keel:"last_verification_status,nullzero"`
	LastVerificationFailedSeq *int64             `json:"lastVerificationFailedSeq" keel:"last_verification_failed_seq"`
	LastVerificationDetail    string             `json:"lastVerificationDetail"    keel:"last_verification_detail,nullzero"`
	VerifyCursorSeq           int64              `json:"verifyCursorSeq"           keel:"verify_cursor_seq"`
	Version                   int64              `json:"version"                   keel:"version,version"`
	CreatedAt                 int64              `json:"createdAt"                 keel:"created_at,autocreate"`
	UpdatedAt                 int64              `json:"updatedAt"                 keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (h *ChainHead) GetTableName() string {
	return "audit_chain_heads"
}

type ClassDigest struct {
	Count  int64  `json:"count"`
	Digest string `json:"digest"`
}

type Seal struct {
	keel.Table `keel:"audit_seals"`

	OrganizationID pulid.ID               `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID               `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Seq            int64                  `json:"seq"            keel:"seq,pk"`
	FromTS         int64                  `json:"fromTs"         keel:"from_ts"`
	ToTS           int64                  `json:"toTs"           keel:"to_ts"`
	RowCount       int64                  `json:"rowCount"       keel:"row_count"`
	ClassDigests   map[string]ClassDigest `json:"classDigests"   keel:"class_digests,jsonb"`
	PrevHash       string                 `json:"prevHash"       keel:"prev_hash"`
	Hash           string                 `json:"hash"           keel:"hash"`
	HashKeyID      string                 `json:"hashKeyId"      keel:"hash_key_id,nullzero"`
	HashVersion    int16                  `json:"hashVersion"    keel:"hash_version"`
	SealedAt       int64                  `json:"sealedAt"       keel:"sealed_at"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (s *Seal) GetTableName() string {
	return "audit_seals"
}
