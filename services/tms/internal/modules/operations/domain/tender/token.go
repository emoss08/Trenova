package tender

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type TenderOfferToken struct {
	keel.Table `keel:"tender_offer_tokens"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:tot_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	TenderOfferID  pulid.ID `json:"tenderOfferId" keel:"tender_offer_id"`

	TokenHash string `json:"-" keel:"token_hash"`
	Email     string `json:"email" keel:"email"`
	ExpiresAt int64  `json:"expiresAt" keel:"expires_at"`
	UsedAt    *int64 `json:"usedAt" keel:"used_at"`
	RevokedAt *int64 `json:"revokedAt" keel:"revoked_at"`
	CreatedAt int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	TenderOffer *TenderOffer `json:"tenderOffer,omitempty" keel:"belongs:tender_offer_id"`
}

func (tot *TenderOfferToken) IsUsable(now int64) bool {
	return tot != nil &&
		tot.UsedAt == nil &&
		tot.RevokedAt == nil &&
		tot.ExpiresAt > now
}

func (tot *TenderOfferToken) GetID() pulid.ID {
	return tot.ID
}

func (tot *TenderOfferToken) GetCreatedAt() int64 {
	return tot.CreatedAt
}

func (tot *TenderOfferToken) GetTableName() string {
	return "tender_offer_tokens"
}

func (tot *TenderOfferToken) GetOrganizationID() pulid.ID {
	return tot.OrganizationID
}

func (tot *TenderOfferToken) GetBusinessUnitID() pulid.ID {
	return tot.BusinessUnitID
}
