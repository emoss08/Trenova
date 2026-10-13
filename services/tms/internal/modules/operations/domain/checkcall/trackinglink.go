package checkcall

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type TrackingLink struct {
	bun.BaseModel `bun:"table:carrier_tracking_links,alias:ctl" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull"`
	TokenHash      string   `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull"`
	RecipientEmail string   `json:"recipientEmail" bun:"recipient_email,type:VARCHAR(320),nullzero"`
	ExpiresAt      int64    `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	RevokedAt      *int64   `json:"revokedAt"      bun:"revoked_at,type:BIGINT,nullzero"`
	LastUsedAt     *int64   `json:"lastUsedAt"     bun:"last_used_at,type:BIGINT,nullzero"`
	CreatedByID    pulid.ID `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`
}

func NewTrackingLinkID() pulid.ID {
	return pulid.MustNew("ctl_")
}

func (l *TrackingLink) IsUsable(now int64) bool {
	return l != nil && l.RevokedAt == nil && l.ExpiresAt > now
}
