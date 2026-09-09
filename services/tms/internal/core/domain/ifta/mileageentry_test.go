package ifta_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validMileageEntry() *ifta.JurisdictionMileageEntry {
	return &ifta.JurisdictionMileageEntry{
		TractorID:      pulid.MustNew("trac_"),
		JurisdictionID: pulid.MustNew("ifj_"),
		TraveledAt:     1_780_000_000,
		Year:           2026,
		Quarter:        2,
		Miles:          decimal.RequireFromString("212.4"),
		Loaded:         true,
		Source:         ifta.MileageSourceManual,
	}
}

func TestJurisdictionMileageEntry_ValidEntryPasses(t *testing.T) {
	t.Parallel()

	entry := validMileageEntry()
	entry.Normalize()

	multiErr := errortypes.NewMultiError()
	entry.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.False(t, entry.OverridesMove())
	assert.Equal(t, ifta.NewPeriod(2026, 2), entry.Period())
	assert.Equal(t, "ifta_jurisdiction_mileage_entries", entry.GetTableName())
}

func TestJurisdictionMileageEntry_ValidateRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(e *ifta.JurisdictionMileageEntry)
		field  string
	}{
		{
			name:   "zero miles",
			mutate: func(e *ifta.JurisdictionMileageEntry) { e.Miles = decimal.Zero },
			field:  "miles",
		},
		{
			name:   "negative miles",
			mutate: func(e *ifta.JurisdictionMileageEntry) { e.Miles = decimal.NewFromInt(-1) },
			field:  "miles",
		},
		{
			name: "over the cap",
			mutate: func(e *ifta.JurisdictionMileageEntry) {
				e.Miles = decimal.RequireFromString("100000.01")
			},
			field: "miles",
		},
		{
			name: "route entry without a move",
			mutate: func(e *ifta.JurisdictionMileageEntry) {
				e.Source = ifta.MileageSourceRouteCalculation
			},
			field: "shipmentMoveId",
		},
		{
			name: "telematics entry without a move",
			mutate: func(e *ifta.JurisdictionMileageEntry) {
				e.Source = ifta.MileageSourceTelematics
			},
			field: "shipmentMoveId",
		},
		{
			name:   "missing tractor",
			mutate: func(e *ifta.JurisdictionMileageEntry) { e.TractorID = pulid.Nil },
			field:  "tractorId",
		},
		{
			name:   "missing jurisdiction",
			mutate: func(e *ifta.JurisdictionMileageEntry) { e.JurisdictionID = pulid.Nil },
			field:  "jurisdictionId",
		},
		{
			name:   "missing travel date",
			mutate: func(e *ifta.JurisdictionMileageEntry) { e.TraveledAt = 0 },
			field:  "traveledAt",
		},
		{
			name:   "bad quarter",
			mutate: func(e *ifta.JurisdictionMileageEntry) { e.Quarter = 0 },
			field:  "quarter",
		},
		{
			name:   "bad source",
			mutate: func(e *ifta.JurisdictionMileageEntry) { e.Source = "GPS" },
			field:  "source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := validMileageEntry()
			tt.mutate(entry)

			multiErr := errortypes.NewMultiError()
			entry.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}

func TestJurisdictionMileageEntry_CapIsInclusive(t *testing.T) {
	t.Parallel()

	entry := validMileageEntry()
	entry.Miles = decimal.NewFromInt(100_000)

	multiErr := errortypes.NewMultiError()
	entry.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestJurisdictionMileageEntry_ManualOverrideOfMovePasses(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	entry := validMileageEntry()
	entry.ShipmentMoveID = &moveID

	multiErr := errortypes.NewMultiError()
	entry.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, entry.OverridesMove())
}

func TestJurisdictionMileageEntry_AssignPeriodUsesLocation(t *testing.T) {
	t.Parallel()

	ny := mustLocation(t, "America/New_York")
	entry := validMileageEntry()
	entry.TraveledAt = time.Date(2026, time.March, 31, 23, 30, 0, 0, ny).Unix()

	entry.AssignPeriod(ny)
	assert.Equal(t, ifta.NewPeriod(2026, 1), entry.Period())

	entry.AssignPeriod(time.UTC)
	assert.Equal(t, ifta.NewPeriod(2026, 2), entry.Period())
}

func TestJurisdictionMileageEntry_NormalizeRoundsAndDefaults(t *testing.T) {
	t.Parallel()

	entry := validMileageEntry()
	entry.Miles = decimal.RequireFromString("12.345")
	entry.Source = ""
	entry.Notes = "  deadhead to yard  "
	entry.Normalize()

	assert.Equal(t, "12.35", entry.Miles.StringFixed(2))
	assert.Equal(t, ifta.MileageSourceManual, entry.Source)
	assert.Equal(t, "deadhead to yard", entry.Notes)
}
