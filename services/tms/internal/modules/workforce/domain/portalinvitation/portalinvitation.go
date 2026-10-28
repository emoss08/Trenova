package portalinvitation

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	_ pagination.CursorEntity            = (*PortalInvitation)(nil)
	_ validationframework.TenantedEntity = (*PortalInvitation)(nil)
)

type PortalInvitationStatus string

const (
	PortalInvitationStatusPending  = PortalInvitationStatus("Pending")
	PortalInvitationStatusAccepted = PortalInvitationStatus("Accepted")
	PortalInvitationStatusRevoked  = PortalInvitationStatus("Revoked")
)

func (s PortalInvitationStatus) String() string { return string(s) }

func (s PortalInvitationStatus) IsValid() bool {
	switch s {
	case PortalInvitationStatusPending, PortalInvitationStatusAccepted,
		PortalInvitationStatusRevoked:
		return true
	default:
		return false
	}
}

type PortalInvitation struct {
	keel.Table     `keel:"worker_portal_invitations"`
	ID             pulid.ID               `keel:"id,pk,autoid:wpi_"          json:"id"`
	BusinessUnitID pulid.ID               `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID pulid.ID               `keel:"organization_id,pk,tenant"  json:"organizationId"`
	WorkerID       pulid.ID               `keel:"worker_id"                  json:"workerId"`
	Email          string                 `keel:"email"                      json:"email"`
	TokenHash      string                 `keel:"token_hash"                 json:"-"`
	Status         PortalInvitationStatus `keel:"status"                     json:"status"`
	ExpiresAt      int64                  `keel:"expires_at"                 json:"expiresAt"`
	InvitedByID    pulid.ID               `keel:"invited_by_id"              json:"invitedById"`
	AcceptedAt     *int64                 `keel:"accepted_at"                json:"acceptedAt"`
	AcceptedUserID *pulid.ID              `keel:"accepted_user_id"           json:"acceptedUserId"`
	Version        int64                  `keel:"version,version"            json:"version"`
	CreatedAt      int64                  `keel:"created_at,autocreate"      json:"createdAt"`
	UpdatedAt      int64                  `keel:"updated_at,autoupdate"      json:"updatedAt"`

	Worker    *worker.Worker `json:"worker,omitempty"    keel:"belongs:worker_id"`
	InvitedBy *tenant.User   `json:"invitedBy,omitempty" keel:"belongs:invited_by_id"`
}

func (i *PortalInvitation) BeforeCreate(_ context.Context) error {
	if i.Status == "" {
		i.Status = PortalInvitationStatusPending
	}

	return nil
}

func (i *PortalInvitation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.WorkerID,
			validation.Required.Error("Worker is required"),
		),
		validation.Field(&i.Email,
			validation.Required.Error("Email is required"),
			is.EmailFormat.Error("Email must be a valid email address"),
		),
		validation.Field(&i.TokenHash,
			validation.Required.Error("Token hash is required"),
		),
		validation.Field(&i.ExpiresAt,
			validation.Required.Error("Expiration is required"),
		),
		validation.Field(&i.InvitedByID,
			validation.Required.Error("Inviting user is required"),
		),
	))

	if !i.Status.IsValid() {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Status must be Pending, Accepted, or Revoked",
		)
	}
}

func (i *PortalInvitation) IsExpired(now int64) bool {
	return i.Status == PortalInvitationStatusPending && now >= i.ExpiresAt
}

func (i *PortalInvitation) IsAcceptable(now int64) bool {
	return i.Status == PortalInvitationStatusPending && now < i.ExpiresAt
}

func (i *PortalInvitation) GetID() pulid.ID { return i.ID }

func (i *PortalInvitation) GetCreatedAt() int64 { return i.CreatedAt }

func (i *PortalInvitation) GetOrganizationID() pulid.ID { return i.OrganizationID }

func (i *PortalInvitation) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }

func (i *PortalInvitation) GetTableName() string { return "worker_portal_invitations" }
