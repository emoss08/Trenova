package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*ApprovalDelegation)(nil)
	_ validationframework.TenantedEntity = (*ApprovalDelegation)(nil)
)

// ApprovalDelegation is one person approving in another's place for a while.
//
// It widens what the delegate may act on; it never widens what the delegator
// could approve in the first place. Delegating from somebody who manages
// nobody hands over nothing, which is the correct outcome rather than a bug.
type ApprovalDelegation struct {
	bun.BaseModel `bun:"table:approval_delegations,alias:apdl" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	DelegatorID pulid.ID      `json:"delegatorId" bun:"delegator_id,type:VARCHAR(100),notnull"`
	DelegateID  pulid.ID      `json:"delegateId"  bun:"delegate_id,type:VARCHAR(100),notnull"`
	Scope       ApprovalScope `json:"scope"       bun:"scope,type:approval_scope_enum,notnull,default:'All'"`

	StartsAt int64  `json:"startsAt" bun:"starts_at,type:BIGINT,notnull"`
	EndsAt   *int64 `json:"endsAt"   bun:"ends_at,type:BIGINT,nullzero"`
	Reason   string `json:"reason"   bun:"reason,type:VARCHAR(255),nullzero"`

	// RevokedAt is set when a delegation is called back before its window ends.
	// It is kept rather than deleted so an approval made under it can still be
	// explained afterwards.
	RevokedAt   *int64   `json:"revokedAt"   bun:"revoked_at,type:BIGINT,nullzero"`
	RevokedByID pulid.ID `json:"revokedById" bun:"revoked_by_id,type:VARCHAR(100),nullzero"`
	CreatedByID pulid.ID `json:"createdById" bun:"created_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Delegator *tenant.User `json:"delegator,omitempty" bun:"rel:belongs-to,join:delegator_id=id"`
	Delegate  *tenant.User `json:"delegate,omitempty"  bun:"rel:belongs-to,join:delegate_id=id"`
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

func (d *ApprovalDelegation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("apdl_")
		}
		if d.Scope == "" {
			d.Scope = ApprovalScopeAll
		}
		if d.StartsAt <= 0 {
			d.StartsAt = now
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}
