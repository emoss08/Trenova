/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package organization

import (
	"context"

	businessunit "github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/common"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*DataRetention)(nil)
	_ common.VersionedEntity    = (*DataRetention)(nil)
)

type DataRetention struct {
	bun.BaseModel `bun:"table:data_retention,alias:dr" json:"-"`

	ID                   pulid.ID `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	AuditRetentionPeriod int      `json:"auditRetentionPeriod" bun:"audit_retention_period,type:INTEGER,notnull,default:120"` // In days
	Version              int64    `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64    `json:"createdAt"            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64    `json:"updatedAt"            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization              `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dr *DataRetention) GetID() string {
	return dr.ID.String()
}

func (dr *DataRetention) GetTableName() string {
	return "data_retention"
}

func (dr *DataRetention) GetVersion() int64 {
	return dr.Version
}

func (dr *DataRetention) IncrementVersion() {
	dr.Version++
}

func (dr *DataRetention) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dr.ID.IsNil() {
			dr.ID = pulid.MustNew("dr_")
		}

		dr.CreatedAt = now
	case *bun.UpdateQuery:
		dr.UpdatedAt = now
	}

	return nil
}
