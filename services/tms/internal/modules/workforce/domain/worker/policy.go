package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var ErrInvalidPolicyAudience = errors.New("invalid policy audience")

// PolicyAudience is who a policy binds. A handbook for employees is not a
// contract term for an owner-operator, and asking a contractor to sign one
// blurs a line the carrier's lawyer would rather keep sharp.
type PolicyAudience string

const (
	PolicyAudienceAll         = PolicyAudience("All")
	PolicyAudienceEmployees   = PolicyAudience("Employees")
	PolicyAudienceContractors = PolicyAudience("Contractors")
)

func (a PolicyAudience) String() string { return string(a) }

func (a PolicyAudience) IsValid() bool {
	switch a {
	case PolicyAudienceAll, PolicyAudienceEmployees, PolicyAudienceContractors:
		return true
	default:
		return false
	}
}

// Covers reports whether a worker of the given type is bound by the policy.
func (a PolicyAudience) Covers(workerType WorkerType) bool {
	switch a {
	case PolicyAudienceAll:
		return true
	case PolicyAudienceEmployees:
		return workerType == WorkerTypeEmployee
	case PolicyAudienceContractors:
		return workerType == WorkerTypeContractor
	default:
		return false
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerPolicy)(nil)
	_ validationframework.TenantedEntity = (*WorkerPolicy)(nil)
)

// WorkerPolicy is a handbook, a policy, or a notice the carrier needs people
// to have read. It is either a short text or an attached document — never
// neither, because a policy that says nothing is a signature with nothing
// behind it.
//
// The version label is the load-bearing field. An acknowledgement records the
// version it was given for, so changing the text under a signed policy without
// changing the version is refused: it would make every existing signature a
// signature on words nobody saw.
type WorkerPolicy struct {
	bun.BaseModel `bun:"table:worker_policies,alias:wpol" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Status            domaintypes.Status `json:"status"            bun:"status,type:status_enum,notnull,default:'Active'"`
	Code              string             `json:"code"              bun:"code,type:VARCHAR(30),notnull"`
	Title             string             `json:"title"             bun:"title,type:VARCHAR(150),notnull"`
	Summary           string             `json:"summary"           bun:"summary,type:TEXT,nullzero"`
	Body              string             `json:"body"              bun:"body,type:TEXT,nullzero"`
	DocumentID        pulid.ID           `json:"documentId"        bun:"document_id,type:VARCHAR(100),nullzero"`
	VersionLabel      string             `json:"versionLabel"      bun:"version_label,type:VARCHAR(30),notnull,default:'1'"`
	RequiresSignature bool               `json:"requiresSignature" bun:"requires_signature,type:BOOLEAN,notnull"`
	AppliesTo         PolicyAudience     `json:"appliesTo"         bun:"applies_to,type:worker_policy_audience_enum,notnull,default:'All'"`
	EffectiveFrom     int64              `json:"effectiveFrom"     bun:"effective_from,type:BIGINT,notnull"`
	CreatedByID       pulid.ID           `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// HasContent reports whether there is anything to sign.
func (p *WorkerPolicy) HasContent() bool {
	return strings.TrimSpace(p.Body) != "" || !p.DocumentID.IsNil()
}

// ContentChanged reports whether the words behind the policy differ from an
// earlier copy. The title and summary are not words anybody signed.
func (p *WorkerPolicy) ContentChanged(from *WorkerPolicy) bool {
	if from == nil {
		return false
	}
	return strings.TrimSpace(p.Body) != strings.TrimSpace(from.Body) ||
		p.DocumentID != from.DocumentID
}

func (p *WorkerPolicy) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Code,
			validation.Required.Error("A code is required"),
			validation.Length(1, 30).Error("Code cannot exceed 30 characters"),
		),
		validation.Field(&p.Title,
			validation.Required.Error("A title is required"),
			validation.Length(1, 150).Error("Title cannot exceed 150 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
		validation.Field(&p.VersionLabel,
			validation.Required.Error("A version is required"),
			validation.Length(1, 30).Error("Version cannot exceed 30 characters"),
		),
		validation.Field(&p.AppliesTo,
			validation.Required.Error("Who the policy applies to is required"),
			domainvalidation.ValidEnum[PolicyAudience]("Audience is not valid"),
		),
		validation.Field(&p.EffectiveFrom,
			validation.Required.Error("An effective date is required"),
		),
	))

	if !p.HasContent() {
		multiErr.Add(
			"body",
			errortypes.ErrRequired,
			"A policy needs either text or an attached document — a signature has to be on something",
		)
	}
}

func (p *WorkerPolicy) Normalise() {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	p.Title = strings.TrimSpace(p.Title)
	p.Summary = strings.TrimSpace(p.Summary)
	p.Body = strings.TrimSpace(p.Body)
	p.VersionLabel = strings.TrimSpace(p.VersionLabel)
}

func (p *WorkerPolicy) GetID() pulid.ID { return p.ID }

func (p *WorkerPolicy) GetCreatedAt() int64 { return p.CreatedAt }

func (p *WorkerPolicy) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *WorkerPolicy) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *WorkerPolicy) GetTableName() string { return "worker_policies" }

func (p *WorkerPolicy) GetResourceType() string { return "worker_policy" }

func (p *WorkerPolicy) GetResourceID() string { return p.ID.String() }

func (p *WorkerPolicy) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("wpol_")
		}
		if p.Status == "" {
			p.Status = domaintypes.StatusActive
		}
		if p.VersionLabel == "" {
			p.VersionLabel = "1"
		}
		if p.AppliesTo == "" {
			p.AppliesTo = PolicyAudienceAll
		}
		if p.EffectiveFrom <= 0 {
			p.EffectiveFrom = now
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerPolicyAcknowledgement)(nil)
	_ validationframework.TenantedEntity = (*WorkerPolicyAcknowledgement)(nil)
)

// WorkerPolicyAcknowledgement is one worker's signature on one version of a
// policy. It copies what was signed — the version label and the document's
// checksum — because the policy row will move on and the signature must not.
//
// The signature is a typed name plus the address and client it came from and
// the moment it was given. That is what an electronic signature is: not a
// picture of handwriting, but a record that a specific person, from a specific
// place, at a specific time, agreed to a specific text.
type WorkerPolicyAcknowledgement struct {
	bun.BaseModel `bun:"table:worker_policy_acknowledgements,alias:wpak" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	PolicyID       pulid.ID `json:"policyId"       bun:"policy_id,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	VersionLabel       string `json:"versionLabel"       bun:"version_label,type:VARCHAR(30),notnull"`
	AcknowledgedAt     int64  `json:"acknowledgedAt"     bun:"acknowledged_at,type:BIGINT,notnull"`
	SignatureName      string `json:"signatureName"      bun:"signature_name,type:VARCHAR(150),nullzero"`
	SignatureIP        string `json:"signatureIp"        bun:"signature_ip,type:VARCHAR(64),nullzero"`
	SignatureUserAgent string `json:"signatureUserAgent" bun:"signature_user_agent,type:VARCHAR(255),nullzero"`
	DocumentChecksum   string `json:"documentChecksum"   bun:"document_checksum,type:VARCHAR(64),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Policy *WorkerPolicy `json:"policy,omitempty" bun:"rel:belongs-to,join:policy_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker *Worker       `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (a *WorkerPolicyAcknowledgement) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.PolicyID, validation.Required.Error("Policy is required")),
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.VersionLabel,
			validation.Required.Error("The version signed is required"),
		),
		validation.Field(&a.AcknowledgedAt,
			validation.Required.Error("The moment of signing is required"),
		),
		validation.Field(&a.SignatureName,
			validation.Length(0, 150).Error("Signature cannot exceed 150 characters"),
		),
	))
}

// Signed reports whether the acknowledgement carries a signature rather than
// a bare read receipt.
func (a *WorkerPolicyAcknowledgement) Signed() bool {
	return strings.TrimSpace(a.SignatureName) != ""
}

func (a *WorkerPolicyAcknowledgement) GetID() pulid.ID { return a.ID }

func (a *WorkerPolicyAcknowledgement) GetCreatedAt() int64 { return a.CreatedAt }

func (a *WorkerPolicyAcknowledgement) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *WorkerPolicyAcknowledgement) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *WorkerPolicyAcknowledgement) GetTableName() string {
	return "worker_policy_acknowledgements"
}

func (a *WorkerPolicyAcknowledgement) GetResourceType() string {
	return "worker_policy_acknowledgement"
}

func (a *WorkerPolicyAcknowledgement) GetResourceID() string { return a.ID.String() }

func (a *WorkerPolicyAcknowledgement) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("wpak_")
		}
		if a.AcknowledgedAt <= 0 {
			a.AcknowledgedAt = now
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}
