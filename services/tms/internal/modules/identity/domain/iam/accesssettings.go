package iam

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*AccessSettings)(nil)

type AccessSettings struct {
	bun.BaseModel `bun:"table:organization_access_settings,alias:oas" json:"-"`

	OrganizationID        pulid.ID `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	RequireRoleActivation bool     `json:"requireRoleActivation" bun:"require_role_activation,type:BOOLEAN,notnull"`
	UpdatedByID           pulid.ID `json:"updatedById"           bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version               int64    `json:"version"               bun:"version,type:BIGINT,notnull"`
	CreatedAt             int64    `json:"createdAt"             bun:"created_at,notnull"`
	UpdatedAt             int64    `json:"updatedAt"             bun:"updated_at,notnull"`
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
