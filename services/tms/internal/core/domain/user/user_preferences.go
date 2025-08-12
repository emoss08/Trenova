package user

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*User)(nil)

type UserPreferences struct {
	bun.BaseModel `bun:"table:user_preferences,alias:up" json:"-"`

	ID                    pulid.ID `json:"id"                    bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),notnull"`
	UserID                pulid.ID `json:"userId"                bun:"user_id,type:VARCHAR(100),notnull"`
	AutoShipmentOwnership bool     `json:"autoShipmentOwnership" bun:"auto_shipment_ownership,type:BOOLEAN,notnull,default:true"` // Automatically assign user as owner of created shipments
	Version               int64    `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt             int64    `json:"createdAt"             bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64    `json:"updatedAt"             bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"` //nolint:lll // this is a long comment

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (up *UserPreferences) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if up.ID.IsNil() {
			up.ID = pulid.MustNew("up_")
		}

		up.CreatedAt = now
	case *bun.UpdateQuery:
		up.UpdatedAt = now
	}

	return nil
}
