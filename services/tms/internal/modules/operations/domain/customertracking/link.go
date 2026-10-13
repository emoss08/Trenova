package customertracking

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type Link struct {
	bun.BaseModel `bun:"table:customer_tracking_links,alias:cstl" json:"-"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID    `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	CustomerID     pulid.ID    `json:"customerId"     bun:"customer_id,type:VARCHAR(100),notnull"`
	TokenHash      string      `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull"`
	Purpose        LinkPurpose `json:"purpose"        bun:"purpose,type:VARCHAR(20),notnull"`
	RecipientEmail string      `json:"recipientEmail" bun:"recipient_email,type:VARCHAR(320),nullzero"`
	ExpiresAt      int64       `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	RevokedAt      *int64      `json:"revokedAt"      bun:"revoked_at,type:BIGINT,nullzero"`
	LastViewedAt   *int64      `json:"lastViewedAt"   bun:"last_viewed_at,type:BIGINT,nullzero"`
	ViewCount      int64       `json:"viewCount"      bun:"view_count,type:BIGINT,notnull,default:0"`
	CreatedByID    pulid.ID    `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64       `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`
}

func NewLinkID() pulid.ID {
	return pulid.MustNew("cstl_")
}

func (l *Link) IsUsable(now int64) bool {
	return l != nil && l.RevokedAt == nil && l.ExpiresAt > now
}
