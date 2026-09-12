package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidCredentialStatus = errors.New("invalid credential status")
	ErrInvalidCredentialHealth = errors.New("invalid credential health")
)

type CredentialStatus string

const (
	CredentialStatusActive   = CredentialStatus("Active")
	CredentialStatusArchived = CredentialStatus("Archived")
)

func (s CredentialStatus) String() string { return string(s) }

func (s CredentialStatus) IsValid() bool {
	switch s {
	case CredentialStatusActive, CredentialStatusArchived:
		return true
	default:
		return false
	}
}

// CredentialHealth is the evaluated state of one credential slot for a worker.
// Missing is only produced by the summary: a credential row that exists is
// never Missing.
type CredentialHealth string

const (
	CredentialHealthValid        = CredentialHealth("Valid")
	CredentialHealthExpiringSoon = CredentialHealth("ExpiringSoon")
	CredentialHealthExpired      = CredentialHealth("Expired")
	CredentialHealthMissing      = CredentialHealth("Missing")
)

func (h CredentialHealth) String() string { return string(h) }

func (h CredentialHealth) IsValid() bool {
	switch h {
	case CredentialHealthValid, CredentialHealthExpiringSoon, CredentialHealthExpired,
		CredentialHealthMissing:
		return true
	default:
		return false
	}
}

// Blocks reports whether this health makes a required credential unusable.
func (h CredentialHealth) Blocks() bool {
	return h == CredentialHealthExpired || h == CredentialHealthMissing
}

const secondsPerDay = int64(86400)

// DaysUntil returns whole calendar days from now until expiry, negative once
// expired. Both instants are treated as UTC dates so the sweep, the summary and
// the client agree on the day a credential turns.
func DaysUntil(expiresAt, now int64) int64 {
	expiryDay := expiresAt / secondsPerDay
	if expiresAt < 0 && expiresAt%secondsPerDay != 0 {
		expiryDay--
	}
	nowDay := now / secondsPerDay
	if now < 0 && now%secondsPerDay != 0 {
		nowDay--
	}
	return expiryDay - nowDay
}

// EvaluateCredentialHealth grades an expiry against a renewal window. A nil
// expiry never expires and is always Valid.
func EvaluateCredentialHealth(expiresAt *int64, renewalWindowDays int32, now int64) CredentialHealth {
	if expiresAt == nil || *expiresAt <= 0 {
		return CredentialHealthValid
	}
	days := DaysUntil(*expiresAt, now)
	switch {
	case days < 0:
		return CredentialHealthExpired
	case days <= int64(renewalWindowDays):
		return CredentialHealthExpiringSoon
	default:
		return CredentialHealthValid
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerCredential)(nil)
	_ validationframework.TenantedEntity = (*WorkerCredential)(nil)
)

type WorkerCredential struct {
	bun.BaseModel `bun:"table:worker_credentials,alias:wcred" json:"-"`

	ID               pulid.ID         `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID         `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID         `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID         pulid.ID         `json:"workerId"         bun:"worker_id,type:VARCHAR(100),notnull"`
	CredentialTypeID pulid.ID         `json:"credentialTypeId" bun:"credential_type_id,type:VARCHAR(100),notnull"`
	Status           CredentialStatus `json:"status"           bun:"status,type:worker_credential_status_enum,notnull,default:'Active'"`
	Number           string           `json:"number"           bun:"number,type:VARCHAR(100),nullzero"`
	IssuingAuthority string           `json:"issuingAuthority" bun:"issuing_authority,type:VARCHAR(100),nullzero"`
	IssuedAt         *int64           `json:"issuedAt"         bun:"issued_at,type:BIGINT,nullzero"`
	ExpiresAt        *int64           `json:"expiresAt"        bun:"expires_at,type:BIGINT,nullzero"`
	DocumentID       pulid.ID         `json:"documentId"       bun:"document_id,type:VARCHAR(100),nullzero"`
	Notes            string           `json:"notes"            bun:"notes,type:TEXT,nullzero"`
	VerifiedByID     pulid.ID         `json:"verifiedById"     bun:"verified_by_id,type:VARCHAR(100),nullzero"`
	VerifiedAt       *int64           `json:"verifiedAt"       bun:"verified_at,type:BIGINT,nullzero"`
	ArchivedByID     pulid.ID         `json:"archivedById"     bun:"archived_by_id,type:VARCHAR(100),nullzero"`
	ArchivedAt       *int64           `json:"archivedAt"       bun:"archived_at,type:BIGINT,nullzero"`
	ArchiveReason    string           `json:"archiveReason"    bun:"archive_reason,type:VARCHAR(255),nullzero"`
	Version          int64            `json:"version"          bun:"version,type:BIGINT"`
	CreatedAt        int64            `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64            `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CredentialType *WorkerCredentialType `json:"credentialType,omitempty" bun:"rel:belongs-to,join:credential_type_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker         *Worker               `json:"worker,omitempty"         bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document       *document.Document    `json:"document,omitempty"       bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	VerifiedBy     *tenant.User          `json:"verifiedBy,omitempty"     bun:"rel:belongs-to,join:verified_by_id=id"`
}

func (c *WorkerCredential) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&c.CredentialTypeID,
			validation.Required.Error("Credential type is required"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[CredentialStatus]("status must be Active or Archived"),
		),
		validation.Field(&c.Number,
			validation.Length(0, 100).Error("Number cannot exceed 100 characters"),
		),
		validation.Field(&c.IssuingAuthority,
			validation.Length(0, 100).Error("Issuing authority cannot exceed 100 characters"),
		),
		validation.Field(&c.ArchiveReason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	if c.IssuedAt != nil && *c.IssuedAt <= 0 {
		multiErr.Add("issuedAt", errortypes.ErrInvalid, "Issue date must be a valid date")
	}
	if c.ExpiresAt != nil && *c.ExpiresAt <= 0 {
		multiErr.Add("expiresAt", errortypes.ErrInvalid, "Expiry date must be a valid date")
	}
	if c.IssuedAt != nil && c.ExpiresAt != nil && *c.ExpiresAt <= *c.IssuedAt {
		multiErr.Add("expiresAt", errortypes.ErrInvalid, "Expiry must be after the issue date")
	}
}

// ValidateAgainstType applies the per-type requirements that the row alone
// cannot know about.
func (c *WorkerCredential) ValidateAgainstType(
	credentialType *WorkerCredentialType,
	multiErr *errortypes.MultiError,
) {
	if credentialType == nil {
		return
	}
	if credentialType.RequiresNumber && c.Number == "" {
		multiErr.Add(
			"number",
			errortypes.ErrRequired,
			"{0} requires a credential number", credentialType.Name,
		)
	}
	if credentialType.ProfileField == CredentialProfileFieldLicenseExpiry && c.ExpiresAt == nil {
		multiErr.Add("expiresAt", errortypes.ErrRequired, "A licence must have an expiry date")
	}
}

func (c *WorkerCredential) IsActive() bool { return c.Status == CredentialStatusActive }

func (c *WorkerCredential) IsVerified() bool {
	return c.VerifiedAt != nil && *c.VerifiedAt > 0
}

func (c *WorkerCredential) Health(now int64) CredentialHealth {
	window := int32(30)
	if c.CredentialType != nil {
		window = c.CredentialType.RenewalWindowDays
	}
	return EvaluateCredentialHealth(c.ExpiresAt, window, now)
}

func (c *WorkerCredential) DaysUntilExpiry(now int64) *int64 {
	if c.ExpiresAt == nil || *c.ExpiresAt <= 0 {
		return nil
	}
	days := DaysUntil(*c.ExpiresAt, now)
	return &days
}

func (c *WorkerCredential) GetID() pulid.ID { return c.ID }

func (c *WorkerCredential) GetCreatedAt() int64 { return c.CreatedAt }

func (c *WorkerCredential) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *WorkerCredential) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *WorkerCredential) GetTableName() string { return "worker_credentials" }

func (c *WorkerCredential) GetResourceType() string { return "worker_credential" }

func (c *WorkerCredential) GetResourceID() string { return c.ID.String() }

func (c *WorkerCredential) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("wcred_")
		}
		if c.Status == "" {
			c.Status = CredentialStatusActive
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
