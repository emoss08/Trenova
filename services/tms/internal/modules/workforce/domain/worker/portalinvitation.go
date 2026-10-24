package worker

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
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
	ID             pulid.ID               `json:"id" keel:"id,pk,autoid:wpi_"`
	BusinessUnitID pulid.ID               `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID               `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID               `json:"workerId" keel:"worker_id"`
	Email          string                 `json:"email" keel:"email"`
	TokenHash      string                 `json:"-" keel:"token_hash"`
	Status         PortalInvitationStatus `json:"status" keel:"status"`
	ExpiresAt      int64                  `json:"expiresAt" keel:"expires_at"`
	InvitedByID    pulid.ID               `json:"invitedById" keel:"invited_by_id"`
	AcceptedAt     *int64                 `json:"acceptedAt" keel:"accepted_at"`
	AcceptedUserID *pulid.ID              `json:"acceptedUserId" keel:"accepted_user_id"`
	Version        int64                  `json:"version" keel:"version,version"`
	CreatedAt      int64                  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64                  `json:"updatedAt" keel:"updated_at,autoupdate"`

	Worker    *Worker      `json:"worker,omitempty" keel:"belongs:worker_id"`
	InvitedBy *tenant.User `json:"invitedBy,omitempty" keel:"belongs:invited_by_id"`
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
