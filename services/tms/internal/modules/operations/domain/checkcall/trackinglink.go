package checkcall

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type TrackingLink struct {
	bun.BaseModel `bun:"table:carrier_tracking_links,alias:ctl" json:"-"`
	keel.Table    `keel:"carrier_tracking_links"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),notnull" keel:"shipment_move_id"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull" keel:"carrier_id"`
	TokenHash      string   `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull" keel:"token_hash"`
	RecipientEmail string   `json:"recipientEmail" bun:"recipient_email,type:VARCHAR(320),nullzero" keel:"recipient_email,nullzero"`
	ExpiresAt      int64    `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull" keel:"expires_at"`
	RevokedAt      *int64   `json:"revokedAt"      bun:"revoked_at,type:BIGINT,nullzero" keel:"revoked_at"`
	LastUsedAt     *int64   `json:"lastUsedAt"     bun:"last_used_at,type:BIGINT,nullzero" keel:"last_used_at"`
	CreatedByID    pulid.ID `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id,nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull" keel:"created_at"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull" keel:"updated_at"`
}

func NewTrackingLinkID() pulid.ID {
	return pulid.MustNew("ctl_")
}

func (l *TrackingLink) IsUsable(now int64) bool {
	return l != nil && l.RevokedAt == nil && l.ExpiresAt > now
}
