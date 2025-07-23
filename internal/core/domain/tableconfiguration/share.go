// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package tableconfiguration

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

// ConfigurationShare represents the sharing details of a configuration
type ConfigurationShare struct {
	bun.BaseModel `bun:"table:table_configuration_shares,alias:tcs" json:"-"`

	// Primary identifiers
	ID              pulid.ID  `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	ConfigurationID pulid.ID  `json:"configurationId" bun:"configuration_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID  `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID  `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	SharedWithID    pulid.ID  `json:"sharedWithId"    bun:"shared_with_id,type:VARCHAR(100),notnull"`
	ShareType       ShareType `json:"shareType"       bun:"share_type,type:VARCHAR(20),notnull"`

	// Metadata
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	ShareWithUser *user.User                 `json:"shareWithUser,omitempty" bun:"rel:belongs-to,join:shared_with_id=id"`
	Configuration *Configuration             `json:"configuration,omitempty" bun:"rel:belongs-to,join:configuration_id=id"`
	BusinessUnit  *businessunit.BusinessUnit `json:"businessUnit,omitempty"  bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization  *organization.Organization `json:"organization,omitempty"  bun:"rel:belongs-to,join:organization_id=id"`
}

func (s *ConfigurationShare) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if _, ok := query.(*bun.InsertQuery); ok {
		if s.ID == "" {
			s.ID = pulid.MustNew("tcs_")
		}
		s.CreatedAt = now
	}

	return nil
}
