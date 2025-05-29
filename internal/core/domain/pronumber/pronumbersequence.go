package pronumber

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

const (
	maxInt16 = 32767
	minInt16 = -32768
)

// Sequence represents a pro number sequence for a specific organization and time period
type Sequence struct {
	bun.BaseModel `bun:"table:pro_number_sequences,alias:pns"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100)"                     json:"id"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),notnull" json:"organizationId"`

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

// SafeInt16 converts an int to int16 safely, returning an error if the value would overflow
func SafeInt16(value int) (int16, error) {
	if value > maxInt16 || value < minInt16 {
		return 0, fmt.Errorf("value %d is outside int16 range", value)
	}
	return int16(value), nil
}

// GenerateProNumber generates a new pro number using the sequence
func GenerateProNumber(sequence int64, year, month int) string {
	// Ensure year is two digits and month is 1-12
	twoDigitYear := year % 100
	normalizedMonth := ((month-1)%12+12)%12 + 1 // Ensures month is 1-12
	return fmt.Sprintf("S%02d%02d%011d", twoDigitYear, normalizedMonth, sequence)
}
