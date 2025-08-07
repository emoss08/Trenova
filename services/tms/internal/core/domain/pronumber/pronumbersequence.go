/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package pronumber

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

// Sequence represents a pro number sequence for a specific organization and time period
type Sequence struct {
	bun.BaseModel `bun:"table:pro_number_sequences,alias:pns"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100)"`
	Year            int16    `json:"year"            bun:"year,notnull"`
	Month           int16    `json:"month"           bun:"month,notnull"`
	CurrentSequence int64    `json:"currentSequence" bun:"current_sequence,notnull"`
	Version         int64    `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt       int64    `json:"createdAt"       bun:"created_at,notnull"`
	UpdatedAt       int64    `json:"updatedAt"       bun:"updated_at,notnull"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"businessUnit,omitempty"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"organization,omitempty"`
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (s *Sequence) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("pns_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
