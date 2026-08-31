package driverpay

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func validAssignment() WorkerPayAssignment {
	return WorkerPayAssignment{
		ID:             pulid.MustNew("wpa_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		PayProfileID:   pulid.MustNew("dpp_"),
		EffectiveFrom:  1_700_000_000,
		SplitPercent:   decimal.NewFromInt(100),
	}
}

func TestWorkerPayAssignment_Validate(t *testing.T) {
	t.Parallel()

	endBeforeStart := int64(1_600_000_000)
	endAtStart := int64(1_700_000_000)
	endAfterStart := int64(1_800_000_000)
	componentID := pulid.MustNew("dppc_")

	tests := []struct {
		name    string
		mutate  func(a *WorkerPayAssignment)
		wantErr bool
	}{
		{
			name:    "valid assignment passes",
			mutate:  func(a *WorkerPayAssignment) {},
			wantErr: false,
		},
		{
			name:    "missing worker fails",
			mutate:  func(a *WorkerPayAssignment) { a.WorkerID = "" },
			wantErr: true,
		},
		{
			name:    "missing pay profile fails",
			mutate:  func(a *WorkerPayAssignment) { a.PayProfileID = "" },
			wantErr: true,
		},
		{
			name:    "missing effective from fails",
			mutate:  func(a *WorkerPayAssignment) { a.EffectiveFrom = 0 },
			wantErr: true,
		},
		{
			name:    "effective to before effective from fails",
			mutate:  func(a *WorkerPayAssignment) { a.EffectiveTo = &endBeforeStart },
			wantErr: true,
		},
		{
			name:    "effective to equal to effective from fails",
			mutate:  func(a *WorkerPayAssignment) { a.EffectiveTo = &endAtStart },
			wantErr: true,
		},
		{
			name:    "effective to after effective from passes",
			mutate:  func(a *WorkerPayAssignment) { a.EffectiveTo = &endAfterStart },
			wantErr: false,
		},
		{
			name:    "zero split percent fails",
			mutate:  func(a *WorkerPayAssignment) { a.SplitPercent = decimal.Zero },
			wantErr: true,
		},
		{
			name:    "negative split percent fails",
			mutate:  func(a *WorkerPayAssignment) { a.SplitPercent = decimal.NewFromInt(-1) },
			wantErr: true,
		},
		{
			name:    "split percent above 100 fails",
			mutate:  func(a *WorkerPayAssignment) { a.SplitPercent = decimal.NewFromFloat(100.01) },
			wantErr: true,
		},
		{
			name: "override without component fails",
			mutate: func(a *WorkerPayAssignment) {
				a.RateOverrides = []RateOverride{{Rate: decimal.NewFromFloat(0.5)}}
			},
			wantErr: true,
		},
		{
			name: "duplicate override component fails",
			mutate: func(a *WorkerPayAssignment) {
				a.RateOverrides = []RateOverride{
					{ComponentID: componentID, Rate: decimal.NewFromFloat(0.5)},
					{ComponentID: componentID, Rate: decimal.NewFromFloat(0.6)},
				}
			},
			wantErr: true,
		},
		{
			name: "negative override rate fails",
			mutate: func(a *WorkerPayAssignment) {
				a.RateOverrides = []RateOverride{
					{ComponentID: componentID, Rate: decimal.NewFromInt(-1)},
				}
			},
			wantErr: true,
		},
		{
			name: "valid overrides pass",
			mutate: func(a *WorkerPayAssignment) {
				a.RateOverrides = []RateOverride{
					{
						ComponentID: componentID,
						Rate:        decimal.NewFromFloat(0.62),
					},
					{ComponentID: pulid.MustNew("dppc_"), Rate: decimal.Zero},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entity := validAssignment()
			tt.mutate(&entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestWorkerPayAssignment_OverrideFor(t *testing.T) {
	t.Parallel()

	componentID := pulid.MustNew("dppc_")
	assignment := WorkerPayAssignment{
		RateOverrides: []RateOverride{
			{ComponentID: componentID, Rate: decimal.NewFromFloat(0.62)},
		},
	}

	rate, found := assignment.OverrideFor(componentID)
	assert.True(t, found)
	assert.Equal(t, "0.62", rate.String())

	rate, found = assignment.OverrideFor(pulid.MustNew("dppc_"))
	assert.False(t, found)
	assert.True(t, rate.IsZero())
}

func TestWorkerPayAssignment_IsEffectiveAt(t *testing.T) {
	t.Parallel()

	from := int64(1_700_000_000)
	to := int64(1_800_000_000)

	openEnded := WorkerPayAssignment{EffectiveFrom: from}
	assert.False(t, openEnded.IsEffectiveAt(from-1))
	assert.True(t, openEnded.IsEffectiveAt(from))
	assert.True(t, openEnded.IsEffectiveAt(from+1_000_000_000))

	bounded := WorkerPayAssignment{EffectiveFrom: from, EffectiveTo: &to}
	assert.False(t, bounded.IsEffectiveAt(from-1))
	assert.True(t, bounded.IsEffectiveAt(from))
	assert.True(t, bounded.IsEffectiveAt(to-1))
	assert.False(t, bounded.IsEffectiveAt(to))
	assert.False(t, bounded.IsEffectiveAt(to+1))
}
