// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package consolidation

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

// ConsolidationSequence represents a consolidation sequence for a specific organization and time period
//
//nolint:revive // valid struct name
type ConsolidationSequence struct {
	bun.BaseModel `bun:"table:consolidation_sequences,alias:cs"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`

	// Sequence tracking fields
	Year            int16 `bun:"year,notnull"             json:"year"`
	Month           int16 `bun:"month,notnull"            json:"month"`
	CurrentSequence int64 `bun:"current_sequence,notnull" json:"currentSequence"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,notnull"  json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,notnull"  json:"updatedAt"`
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (s *ConsolidationSequence) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("cs_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
