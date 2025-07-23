// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package sequencestore

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

// Sequence represents a generic sequence counter
type Sequence struct {
	bun.BaseModel `bun:"table:sequences,alias:seq"`

	ID              pulid.ID     `bun:"id,pk,type:VARCHAR(100)"`
	SequenceType    SequenceType `bun:"sequence_type,notnull"`
	OrganizationID  pulid.ID     `bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID     `bun:"business_unit_id,type:VARCHAR(100)"`
	Year            int16        `bun:"year,notnull"`
	Month           int16        `bun:"month,notnull"`
	CurrentSequence int64        `bun:"current_sequence,notnull"`
	Version         int64        `bun:"version,type:BIGINT"`
	CreatedAt       int64        `bun:"created_at,notnull"`
	UpdatedAt       int64        `bun:"updated_at,notnull"`
}

// BeforeAppendModel implements bun hook
func (s *Sequence) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("seq_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
