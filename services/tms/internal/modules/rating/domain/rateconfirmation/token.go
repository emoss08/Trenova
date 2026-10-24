package rateconfirmation

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type RateConfirmationToken struct {
	keel.Table `keel:"rate_confirmation_tokens"`

	ID                 pulid.ID `json:"id" keel:"id,pk,autoid:rct_"`
	BusinessUnitID     pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	RateConfirmationID pulid.ID `json:"rateConfirmationId" keel:"rate_confirmation_id"`

	TokenHash string `json:"-" keel:"token_hash"`
	Email     string `json:"email" keel:"email"`
	ExpiresAt int64  `json:"expiresAt" keel:"expires_at"`
	UsedAt    *int64 `json:"usedAt" keel:"used_at"`
	RevokedAt *int64 `json:"revokedAt" keel:"revoked_at"`
	CreatedAt int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	RateConfirmation *RateConfirmation `json:"rateConfirmation,omitempty" keel:"belongs:rate_confirmation_id"`
}

func (rct *RateConfirmationToken) IsUsable(now int64) bool {
	return rct != nil &&
		rct.UsedAt == nil &&
		rct.RevokedAt == nil &&
		rct.ExpiresAt > now
}

func (rct *RateConfirmationToken) GetID() pulid.ID {
	return rct.ID
}

func (rct *RateConfirmationToken) GetCreatedAt() int64 {
	return rct.CreatedAt
}

func (rct *RateConfirmationToken) GetTableName() string {
	return "rate_confirmation_tokens"
}

func (rct *RateConfirmationToken) GetOrganizationID() pulid.ID {
	return rct.OrganizationID
}

func (rct *RateConfirmationToken) GetBusinessUnitID() pulid.ID {
	return rct.BusinessUnitID
}
