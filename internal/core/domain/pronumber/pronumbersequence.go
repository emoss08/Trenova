package pronumber

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

// Sequence represents a pro number sequence for a specific organization and time period
type Sequence struct {
	bun.BaseModel `bun:"table:pro_number_sequences,alias:pns"`

	ID              pulid.ID `bun:",pk,type:VARCHAR(100)"                     json:"id"`
	OrganizationID  pulid.ID `bun:"organization_id,type:VARCHAR(100),notnull" json:"organizationId"`
	BusinessUnitID  pulid.ID `bun:"business_unit_id,type:VARCHAR(100)"        json:"businessUnitId"`
	Year            int16    `bun:"year,notnull"                              json:"year"`
	Month           int16    `bun:"month,notnull"                             json:"month"`
	CurrentSequence int64    `bun:"current_sequence,notnull"                  json:"currentSequence"`
	Version         int64    `bun:"version,type:BIGINT"                       json:"version"`
	CreatedAt       int64    `bun:"created_at,notnull"                        json:"createdAt"`
	UpdatedAt       int64    `bun:"updated_at,notnull"                        json:"updatedAt"`

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
