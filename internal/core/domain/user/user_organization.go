// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package user

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*UserOrganization)(nil)

type UserOrganization struct { //nolint:revive // This is a domain object
	bun.BaseModel `bun:"table:user_organizations,alias:uo" json:"-"`

	UserID         pulid.ID `json:"userId"         bun:"user_id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100)"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	User         *User                      `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (uo *UserOrganization) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().Unix()

	if _, ok := query.(*bun.InsertQuery); ok {
		uo.CreatedAt = now
	}

	return nil
}
