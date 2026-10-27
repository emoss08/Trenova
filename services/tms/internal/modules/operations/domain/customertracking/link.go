package customertracking

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type Link struct {
	keel.Table `keel:"customer_tracking_links"`

	ID             pulid.ID    `json:"id"             keel:"id,pk"`
	OrganizationID pulid.ID    `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ShipmentID     pulid.ID    `json:"shipmentId"     keel:"shipment_id"`
	CustomerID     pulid.ID    `json:"customerId"     keel:"customer_id"`
	TokenHash      string      `json:"-"              keel:"token_hash"`
	Purpose        LinkPurpose `json:"purpose"        keel:"purpose"`
	RecipientEmail string      `json:"recipientEmail" keel:"recipient_email,nullzero"`
	ExpiresAt      int64       `json:"expiresAt"      keel:"expires_at"`
	RevokedAt      *int64      `json:"revokedAt"      keel:"revoked_at"`
	LastViewedAt   *int64      `json:"lastViewedAt"   keel:"last_viewed_at"`
	ViewCount      int64       `json:"viewCount"      keel:"view_count"`
	CreatedByID    pulid.ID    `json:"createdById"    keel:"created_by_id,nullzero"`
	CreatedAt      int64       `json:"createdAt"      keel:"created_at"`
	UpdatedAt      int64       `json:"updatedAt"      keel:"updated_at"`
}

func NewLinkID() pulid.ID {
	return pulid.MustNew("cstl_")
}

func (l *Link) IsUsable(now int64) bool {
	return l != nil && l.RevokedAt == nil && l.ExpiresAt > now
}

func (Link) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_customer_tracking_links_shipment",
			Message: "Shipment does not exist in your organization",
		},
	}
}
