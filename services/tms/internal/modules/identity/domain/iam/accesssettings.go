package iam

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*AccessSettings)(nil)

type AccessSettings struct {
	bun.BaseModel `bun:"table:organization_access_settings,alias:oas" json:"-"`
	keel.Table    `keel:"organization_access_settings"`

	OrganizationID        pulid.ID `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	RequireRoleActivation bool     `json:"requireRoleActivation" bun:"require_role_activation,type:BOOLEAN,notnull" keel:"require_role_activation"`
	UpdatedByID           pulid.ID `json:"updatedById"           bun:"updated_by_id,type:VARCHAR(100),nullzero" keel:"updated_by_id,nullzero"`
	Version               int64    `json:"version"               bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt             int64    `json:"createdAt"             bun:"created_at,notnull" keel:"created_at,autocreate"`
	UpdatedAt             int64    `json:"updatedAt"             bun:"updated_at,notnull" keel:"updated_at,autoupdate"`
}

func DefaultAccessSettings(orgID, buID pulid.ID) *AccessSettings {
	return &AccessSettings{OrganizationID: orgID, BusinessUnitID: buID}
}

func (s *AccessSettings) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()
	switch q.(type) {
	case *bun.InsertQuery:
		if s.CreatedAt == 0 {
			s.CreatedAt = now
		}
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}
