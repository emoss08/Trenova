package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var ErrInvalidApprovalScope = errors.New("invalid approval scope")

// ApprovalScope is what a delegation covers. A manager going on holiday
// usually hands over everything; one handing a specific queue to a specialist
// hands over one thing, and a delegation that quietly covered more than it
// said would be worse than no delegation at all.
type ApprovalScope string

const (
	ApprovalScopeAll      = ApprovalScope("All")
	ApprovalScopeTimeOff  = ApprovalScope("TimeOff")
	ApprovalScopeExpenses = ApprovalScope("Expenses")
)

func (s ApprovalScope) String() string { return string(s) }

func (s ApprovalScope) IsValid() bool {
	switch s {
	case ApprovalScopeAll, ApprovalScopeTimeOff, ApprovalScopeExpenses:
		return true
	default:
		return false
	}
}

// Covers reports whether a delegation of this scope answers for the thing
// being approved. All covers everything; anything else covers only itself.
func (s ApprovalScope) Covers(wanted ApprovalScope) bool {
	return s == ApprovalScopeAll || s == wanted
}

var (
	_ validationframework.TenantedEntity = (*ApprovalDelegation)(nil)
)

// ApprovalDelegation is one person approving in another's place for a while.
//
// It widens what the delegate may act on; it never widens what the delegator
// could approve in the first place. Delegating from somebody who manages
// nobody hands over nothing, which is the correct outcome rather than a bug.
type ApprovalDelegation struct {
	keel.Table `                                                     keel:"approval_delegations"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:apdl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	DelegatorID pulid.ID      `json:"delegatorId" keel:"delegator_id"`
	DelegateID  pulid.ID      `json:"delegateId" keel:"delegate_id"`
	Scope       ApprovalScope `json:"scope" keel:"scope"`

	StartsAt int64  `json:"startsAt" keel:"starts_at"`
	EndsAt   *int64 `json:"endsAt" keel:"ends_at"`
	Reason   string `json:"reason" keel:"reason,nullzero"`

	// RevokedAt is set when a delegation is called back before its window ends.
	// It is kept rather than deleted so an approval made under it can still be
	// explained afterwards.
	RevokedAt   *int64   `json:"revokedAt" keel:"revoked_at"`
	RevokedByID pulid.ID `json:"revokedById" keel:"revoked_by_id,nullzero"`
	CreatedByID pulid.ID `json:"createdById" keel:"created_by_id,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Delegator *tenant.User `json:"delegator,omitempty" keel:"belongs:delegator_id"`
	Delegate  *tenant.User `json:"delegate,omitempty" keel:"belongs:delegate_id"`
}

func (d *ApprovalDelegation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.DelegatorID, validation.Required.Error("Delegator is required")),
		validation.Field(&d.DelegateID, validation.Required.Error("Delegate is required")),
		validation.Field(&d.Scope,
			validation.Required.Error("Scope is required"),
			domainvalidation.ValidEnum[ApprovalScope]("Scope is not valid"),
		),
		validation.Field(&d.StartsAt, validation.Required.Error("A start date is required")),
		validation.Field(&d.Reason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	// Delegating to yourself is a no-op that reads on the list as cover somebody
	// arranged, which is worse than nothing while they are away.
	if d.DelegatorID == d.DelegateID {
		multiErr.Add(
			"delegateId",
			errortypes.ErrInvalid,
			"A delegation has to hand approval to somebody else",
		)
	}
	if d.EndsAt != nil && *d.EndsAt < d.StartsAt {
		multiErr.Add("endsAt", errortypes.ErrInvalid, "A delegation cannot end before it begins")
	}
}

// IsActive reports whether the delegation answers at this instant. An open
// end date means it runs until somebody revokes it.
func (d *ApprovalDelegation) IsActive(now int64) bool {
	if d.RevokedAt != nil && *d.RevokedAt > 0 && *d.RevokedAt <= now {
		return false
	}
	if now < d.StartsAt {
		return false
	}
	if d.EndsAt != nil && *d.EndsAt > 0 && now > *d.EndsAt {
		return false
	}
	return true
}

// Normalise trims the free text.
func (d *ApprovalDelegation) Normalise() {
	d.Reason = strings.TrimSpace(d.Reason)
}

func (d *ApprovalDelegation) GetID() pulid.ID { return d.ID }

func (d *ApprovalDelegation) GetCreatedAt() int64 { return d.CreatedAt }

func (d *ApprovalDelegation) GetOrganizationID() pulid.ID { return d.OrganizationID }

func (d *ApprovalDelegation) GetBusinessUnitID() pulid.ID { return d.BusinessUnitID }

func (d *ApprovalDelegation) GetTableName() string { return "approval_delegations" }

func (d *ApprovalDelegation) GetResourceType() string { return "approval_delegation" }

func (d *ApprovalDelegation) GetResourceID() string { return d.ID.String() }

func (d *ApprovalDelegation) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if d.Scope == "" {
		d.Scope = ApprovalScopeAll
	}
	if d.StartsAt <= 0 {
		d.StartsAt = now
	}

	return nil
}
