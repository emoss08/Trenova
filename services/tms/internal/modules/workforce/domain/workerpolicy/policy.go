package workerpolicy

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
func (a PolicyAudience) Covers(workerType worker.WorkerType) bool {
	switch a {
	case PolicyAudienceAll:
		return true
	case PolicyAudienceEmployees:
		return workerType == worker.WorkerTypeEmployee
	case PolicyAudienceContractors:
		return workerType == worker.WorkerTypeContractor
	default:
		return false
	}
}

var (
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
	keel.Table `keel:"worker_policies"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wpol_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Status            domaintypes.Status `json:"status"            keel:"status"`
	Code              string             `json:"code"              keel:"code"`
	Title             string             `json:"title"             keel:"title"`
	Summary           string             `json:"summary"           keel:"summary,nullzero"`
	Body              string             `json:"body"              keel:"body,nullzero"`
	DocumentID        pulid.ID           `json:"documentId"        keel:"document_id,nullzero"`
	VersionLabel      string             `json:"versionLabel"      keel:"version_label"`
	RequiresSignature bool               `json:"requiresSignature" keel:"requires_signature"`
	AppliesTo         PolicyAudience     `json:"appliesTo"         keel:"applies_to"`
	EffectiveFrom     int64              `json:"effectiveFrom"     keel:"effective_from"`
	CreatedByID       pulid.ID           `json:"createdById"       keel:"created_by_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
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

func (p *WorkerPolicy) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
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

	return nil
}

func (WorkerPolicy) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_worker_policies_code",
			Message: "Policy with this code already exists in your organization",
		},
		{
			Name:    "fk_worker_policies_document",
			Message: "Document does not exist or belongs to a different organization",
		},
	}
}

var (
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
	keel.Table `keel:"worker_policy_acknowledgements"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wpak_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	PolicyID       pulid.ID `json:"policyId"       keel:"policy_id"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`

	VersionLabel       string `json:"versionLabel"       keel:"version_label"`
	AcknowledgedAt     int64  `json:"acknowledgedAt"     keel:"acknowledged_at"`
	SignatureName      string `json:"signatureName"      keel:"signature_name,nullzero"`
	SignatureIP        string `json:"signatureIp"        keel:"signature_ip,nullzero"`
	SignatureUserAgent string `json:"signatureUserAgent" keel:"signature_user_agent,nullzero"`
	DocumentChecksum   string `json:"documentChecksum"   keel:"document_checksum,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Policy *WorkerPolicy  `json:"policy,omitempty" keel:"belongs:policy_id"`
	Worker *worker.Worker `json:"worker,omitempty" keel:"belongs:worker_id"`
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

func (a *WorkerPolicyAcknowledgement) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if a.AcknowledgedAt <= 0 {
		a.AcknowledgedAt = now
	}

	return nil
}

func (WorkerPolicyAcknowledgement) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_worker_policy_acknowledgements_version",
			Field:   "versionLabel",
			Message: "Worker has already acknowledged this version of the policy",
		},
		{
			Name:    "fk_worker_policy_acknowledgements_policy",
			Message: "Policy does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_policy_acknowledgements_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
	}
}
