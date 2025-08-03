/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package reportmetadata

import (
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type ReportMetadata struct {
	bun.BaseModel `bun:"table:report_metadata,alias:rpt" json:"-"`

	// Primary Identifiers
	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`

	// Core Attributes
	Name              string            `json:"name"              bun:"name,type:VARCHAR(150),notnull"`
	Description       string            `json:"description"       bun:"description,type:VARCHAR(500),notnull"`
	VisualizationType VisualizationType `json:"visualizationType" bun:"visualization_type,type:VARCHAR(50),default:'Table'"` // Table, Chart, etc.
	Tags              []string          `json:"tags"              bun:"tags,type:TEXTARRAY"`

	// User and System Metadata
	CreatedBy       pulid.ID `json:"createdBy"       bun:"created_by,type:VARCHAR(100),notnull"`
	IsSystemDefined bool     `json:"isSystemDefined" bun:"is_system_defined,type:BOOLEAN,notnull,default:false"`

	// Versioning and Scheduling
	IsScheduled  bool   `json:"isScheduled"  bun:"is_scheduled,type:BOOLEAN,notnull,default:false"`
	ScheduleCron string `json:"scheduleCron" bun:"schedule_cron,type:VARCHAR(100)"` // CRON expression

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}
