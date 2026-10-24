package checkcall

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type TrackingLink struct {
	keel.Table `keel:"carrier_tracking_links"`

	ID             pulid.ID `json:"id" keel:"id,pk"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId" keel:"shipment_id"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" keel:"shipment_move_id"`
	CarrierID      pulid.ID `json:"carrierId" keel:"carrier_id"`
	TokenHash      string   `json:"-" keel:"token_hash"`
	RecipientEmail string   `json:"recipientEmail" keel:"recipient_email,nullzero"`
	ExpiresAt      int64    `json:"expiresAt" keel:"expires_at"`
	RevokedAt      *int64   `json:"revokedAt" keel:"revoked_at"`
	LastUsedAt     *int64   `json:"lastUsedAt" keel:"last_used_at"`
	CreatedByID    pulid.ID `json:"createdById" keel:"created_by_id,nullzero"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at"`
}

func NewTrackingLinkID() pulid.ID {
	return pulid.MustNew("ctl_")
}

func (l *TrackingLink) IsUsable(now int64) bool {
	return l != nil && l.RevokedAt == nil && l.ExpiresAt > now
}
