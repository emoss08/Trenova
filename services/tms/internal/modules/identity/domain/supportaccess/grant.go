package supportaccess

import (
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	GrantIDPrefix   = "sag_"
	MaxLiveGrants   = 5
	MaxReasonLength = 500
)

var (
	ErrGrantNotFound = errors.New("support access grant not found")
	ErrNoKeySet      = errors.New("the control plane has not delivered support access keys yet")
)

type Grant struct {
	keel.Table `keel:"support_access_grants"`

	ID               pulid.ID `json:"id"               keel:"id,pk,autoid:sag_"`
	OrganizationID   pulid.ID `json:"organizationId"   keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	StaffEmail       string   `json:"staffEmail"       keel:"staff_email"`
	Reason           string   `json:"reason"           keel:"reason"`
	Scope            Scope    `json:"scope"            keel:"scope"`
	DurationHours    int      `json:"durationHours"    keel:"duration_hours"`
	PrincipalUserID  pulid.ID `json:"principalUserId"  keel:"principal_user_id"`
	GrantedByID      pulid.ID `json:"grantedById"      keel:"granted_by_id"`
	ExpiresAt        int64    `json:"expiresAt"        keel:"expires_at"`
	RevokedAt        *int64   `json:"revokedAt"        keel:"revoked_at"`
	RevokedByID      pulid.ID `json:"revokedById"      keel:"revoked_by_id,nullzero"`
	RevokeReason     string   `json:"revokeReason"     keel:"revoke_reason,nullzero"`
	ExpiryRecordedAt *int64   `json:"expiryRecordedAt" keel:"expiry_recorded_at"`
	LastEnteredAt    *int64   `json:"lastEnteredAt"    keel:"last_entered_at"`
	LastStaffName    string   `json:"lastStaffName"    keel:"last_staff_name,nullzero"`
	Version          int64    `json:"version"          keel:"version,version"`
	CreatedAt        int64    `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64    `json:"updatedAt"        keel:"updated_at,autoupdate"`

	GrantedBy *tenant.User `json:"grantedBy,omitempty" keel:"belongs:granted_by_id"`
	RevokedBy *tenant.User `json:"revokedBy,omitempty" keel:"belongs:revoked_by_id"`
}

func (Grant) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{Name: "ck_support_access_grants_staff_email", Field: "staffEmail", Message: "Enter a Trenova staff email address"},
		{Name: "ck_support_access_grants_reason", Field: "reason", Message: "Say why Trenova support needs access, in at most 500 characters"},
		{Name: "ck_support_access_grants_duration", Field: "durationHours", Message: "Choose 1, 4, 24 or 72 hours"},
		{Name: "ck_support_access_grants_scope", Field: "scope", Message: "Choose read-only or full access"},
	}
}

func (g *Grant) LiveAt(now int64) bool {
	return g != nil && g.RevokedAt == nil && g.ExpiresAt > now
}

func (g *Grant) StatusAt(now int64) Status {
	switch {
	case g.RevokedAt != nil:
		return StatusRevoked
	case g.ExpiresAt <= now:
		return StatusExpired
	default:
		return StatusLive
	}
}

// Revoke marks a live grant revoked. The reason is required: the customer's audit log says why
// access ended.
func (g *Grant) Revoke(by pulid.ID, reason string, now int64) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len([]rune(reason)) > MaxReasonLength {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrInvalid,
			"Say why you are ending this access, in at most 500 characters",
		)
	}
	if !g.LiveAt(now) {
		return errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"This support access has already ended",
		)
	}
	g.RevokedAt = &now
	g.RevokedByID = by
	g.RevokeReason = reason
	return nil
}

func (g *Grant) GetID() pulid.ID {
	return g.ID
}

func (g *Grant) GetCreatedAt() int64 {
	return g.CreatedAt
}

func (g *Grant) GetTableName() string {
	return "support_access_grants"
}
