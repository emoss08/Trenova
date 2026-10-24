package tender

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*TenderOfferToken)(nil)

type TenderOfferToken struct {
	bun.BaseModel `bun:"table:tender_offer_tokens,alias:tot" json:"-"`
	keel.Table    `keel:"tender_offer_tokens"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:tot_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	TenderOfferID  pulid.ID `json:"tenderOfferId"  bun:"tender_offer_id,type:VARCHAR(100),notnull" keel:"tender_offer_id"`

	TokenHash string `json:"-"         bun:"token_hash,type:VARCHAR(128),notnull" keel:"token_hash"`
	Email     string `json:"email"     bun:"email,type:VARCHAR(255),notnull" keel:"email"`
	ExpiresAt int64  `json:"expiresAt" bun:"expires_at,type:BIGINT,notnull" keel:"expires_at"`
	UsedAt    *int64 `json:"usedAt"    bun:"used_at,type:BIGINT,nullzero" keel:"used_at"`
	RevokedAt *int64 `json:"revokedAt" bun:"revoked_at,type:BIGINT,nullzero" keel:"revoked_at"`
	CreatedAt int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	TenderOffer *TenderOffer `json:"tenderOffer,omitempty" bun:"rel:belongs-to,join:tender_offer_id=id" keel:"belongs:tender_offer_id"`
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

func (tot *TenderOfferToken) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if tot.ID.IsNil() {
			tot.ID = pulid.MustNew("tot_")
		}
		tot.CreatedAt = now
	case *bun.UpdateQuery:
		tot.UpdatedAt = now
	}

	return nil
}
