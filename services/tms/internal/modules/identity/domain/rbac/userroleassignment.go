package rbac

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type UserRoleAssignment struct {
	keel.Table `keel:"user_role_assignments"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ura_"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,tenant"`
	RoleID         pulid.ID `json:"roleId"         keel:"role_id"`
	ExpiresAt      *int64   `json:"expiresAt"      keel:"expires_at"`
	AssignedBy     pulid.ID `json:"assignedBy"     keel:"assigned_by,nullzero"`
	AssignedAt     int64    `json:"assignedAt"     keel:"assigned_at"`

	Role *Role `json:"role,omitempty" keel:"belongs:role_id"`
}

func (ura *UserRoleAssignment) BeforeCreate(ctx context.Context) error {
	ura.AssignedAt = timeutils.NowUnix()

	return nil
}

func (ura *UserRoleAssignment) GetCreatedAt() int64 {
	return ura.AssignedAt
}

func (ura *UserRoleAssignment) IsExpired() bool {
	if ura.ExpiresAt == nil {
		return false
	}
	return *ura.ExpiresAt < timeutils.NowUnix()
}

func (ura *UserRoleAssignment) GetID() pulid.ID {
	return ura.ID
}

func (a *UserRoleAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.UserID, validation.Required.Error("User is required")),
		validation.Field(&a.RoleID, validation.Required.Error("Role is required")),
		// An assignment that expires before it was made grants nothing, which
		// reads to the user as a permission that silently never applied.
		validation.Field(&a.ExpiresAt,
			validation.Min(a.AssignedAt+1).
				Error("Expiry must be after the assignment was made"),
		),
	))
}
