/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package resource

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ResourceDefinition)(nil)

//nolint:revive // Valid struct name
type ResourceDefinition struct {
	bun.BaseModel `bun:"table:resource_definitions,alias:rd" json:"-"`

	// Primary identifiers
	ID pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100)"`

	// Core fields
	ResourceType       permission.Resource `json:"resourceType"       bun:"resource_type,type:VARCHAR(150),notnull"`
	DisplayName        string              `json:"displayName"        bun:"display_name,type:VARCHAR(100),notnull"`
	TableName          string              `json:"tableName"          bun:"table_name,type:VARCHAR(100),notnull"`
	Description        string              `json:"description"        bun:"description,type:TEXT,notnull"`
	AllowCustomFields  bool                `json:"allowCustomFields"  bun:"allow_custom_fields,type:BOOLEAN,notnull,default:false"`
	AllowAutomations   bool                `json:"allowAutomations"   bun:"allow_automations,type:BOOLEAN,notnull,default:false"`
	AllowNotifications bool                `json:"allowNotifications" bun:"allow_notifications,type:BOOLEAN,notnull,default:false"`

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (rd *ResourceDefinition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rd.ID.IsNil() {
			rd.ID = pulid.MustNew("rd_")
		}

		rd.CreatedAt = now
	case *bun.UpdateQuery:
		rd.UpdatedAt = now
	}

	return nil
}
