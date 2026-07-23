package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*PortalInvitation)(nil)
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
	bun.BaseModel             `bun:"table:worker_portal_invitations,alias:wpi" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                    json:"-"`

	ID             pulid.ID               `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID               `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID               `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID               `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	Email          string                 `json:"email"          bun:"email,type:VARCHAR(255),notnull"`
	TokenHash      string                 `json:"-"              bun:"token_hash,type:VARCHAR(64),notnull"`
	Status         PortalInvitationStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`
	ExpiresAt      int64                  `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	InvitedByID    pulid.ID               `json:"invitedById"    bun:"invited_by_id,type:VARCHAR(100),notnull"`
	AcceptedAt     *int64                 `json:"acceptedAt"     bun:"accepted_at,type:BIGINT,nullzero"`
	AcceptedUserID *pulid.ID              `json:"acceptedUserId" bun:"accepted_user_id,type:VARCHAR(100),nullzero"`
	Version        int64                  `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64                  `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                  `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker    *Worker      `json:"worker,omitempty"    bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	InvitedBy *tenant.User `json:"invitedBy,omitempty" bun:"rel:belongs-to,join:invited_by_id=id"`
}

func (i *PortalInvitation) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(i,
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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

func (i *PortalInvitation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("wpi_")
		}
		i.CreatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}
	return nil
}
