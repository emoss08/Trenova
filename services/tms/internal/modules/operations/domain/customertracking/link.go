package customertracking

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type Link struct {
	bun.BaseModel `bun:"table:customer_tracking_links,alias:cstl" json:"-"`
	keel.Table    `keel:"customer_tracking_links"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	ShipmentID     pulid.ID    `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	CustomerID     pulid.ID    `json:"customerId"     bun:"customer_id,type:VARCHAR(100),notnull" keel:"customer_id"`
	TokenHash      string      `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull" keel:"token_hash"`
	Purpose        LinkPurpose `json:"purpose"        bun:"purpose,type:VARCHAR(20),notnull" keel:"purpose"`
	RecipientEmail string      `json:"recipientEmail" bun:"recipient_email,type:VARCHAR(320),nullzero" keel:"recipient_email,nullzero"`
	ExpiresAt      int64       `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull" keel:"expires_at"`
	RevokedAt      *int64      `json:"revokedAt"      bun:"revoked_at,type:BIGINT,nullzero" keel:"revoked_at"`
	LastViewedAt   *int64      `json:"lastViewedAt"   bun:"last_viewed_at,type:BIGINT,nullzero" keel:"last_viewed_at"`
	ViewCount      int64       `json:"viewCount"      bun:"view_count,type:BIGINT,notnull,default:0" keel:"view_count"`
	CreatedByID    pulid.ID    `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id,nullzero"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,type:BIGINT,notnull" keel:"created_at"`
	UpdatedAt      int64       `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull" keel:"updated_at"`
}

func NewLinkID() pulid.ID {
	return pulid.MustNew("cstl_")
}

func (l *Link) IsUsable(now int64) bool {
	return l != nil && l.RevokedAt == nil && l.ExpiresAt > now
}
