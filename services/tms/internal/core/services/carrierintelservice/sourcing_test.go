package carrierintelservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sourcingEntry(dot string, score float64, powerUnits, authorityAge *int) *SourcingResult {
	profile := &carrierintel.Profile{
		Identity: &carrierintel.Identity{DOTNumber: dot},
	}
	if powerUnits != nil {
		profile.Fleet = &carrierintel.Fleet{PowerUnits: powerUnits}
	}
	if authorityAge != nil {
		profile.Authority = &carrierintel.Authority{
			Common: &carrierintel.AuthorityGrant{
				Status:  carrierintel.AuthorityStatusActive,
				AgeDays: authorityAge,
			},
		}
	}
	return &SourcingResult{Profile: profile, DOTNumber: dot, Score: score}
}

func sourcingDOTs(items []*SourcingResult) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.DOTNumber)
	}
	return out
}

func TestSortSourcingResults(t *testing.T) {
	t.Parallel()

	build := func() []*SourcingResult {
		return []*SourcingResult{
			sourcingEntry("1", 70, new(10), new(400)),
			sourcingEntry("2", 95, nil, new(90)),
			sourcingEntry("3", 80, new(250), nil),
			sourcingEntry("4", 60, new(250), new(2000)),
		}
	}

	tests := []struct {
		name  string
		order SourcingSort
		want  []string
	}{
		{name: "best match", order: SourcingSortBestMatch, want: []string{"2", "3", "1", "4"}},
		{name: "unset falls back to score", order: "", want: []string{"2", "3", "1", "4"}},
		{
			name:  "fleet size descending ties on score and puts unknown last",
			order: SourcingSortFleetSizeDesc,
			want:  []string{"3", "4", "1", "2"},
		},
		{
			name:  "authority age descending puts unknown last",
			order: SourcingSortAuthorityAgeDesc,
			want:  []string{"4", "1", "2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			items := build()
			sortSourcingResults(items, tt.order)
			assert.Equal(t, tt.want, sourcingDOTs(items))
		})
	}
}

func TestMatchesSourcingFiltersAuthorityAgeRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		age  *int
		min  *int
		max  *int
		want bool
	}{
		{name: "no bounds", age: nil, want: true},
		{name: "within range", age: new(365), min: new(180), max: new(730), want: true},
		{name: "below minimum", age: new(90), min: new(180), want: false},
		{name: "above maximum", age: new(900), max: new(730), want: false},
		{name: "on the maximum", age: new(730), max: new(730), want: true},
		{name: "unknown age with a maximum", age: nil, max: new(730), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := sourcingEntry("1", 0, nil, tt.age)
			query := &SourcingQuery{MinAuthorityAgeDays: tt.min, MaxAuthorityAgeDays: tt.max}
			assert.Equal(t, tt.want, matchesSourcingFilters(entry.Profile, query))
		})
	}
}

func TestValidateSourcingQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query SourcingQuery
		field string
	}{
		{
			name:  "valid",
			query: SourcingQuery{MinAuthorityAgeDays: new(10), MaxAuthorityAgeDays: new(10)},
		},
		{
			name:  "authority age range inverted",
			query: SourcingQuery{MinAuthorityAgeDays: new(400), MaxAuthorityAgeDays: new(100)},
			field: "maxAuthorityAgeDays",
		},
		{
			name:  "negative maximum authority age",
			query: SourcingQuery{MaxAuthorityAgeDays: new(-1)},
			field: "maxAuthorityAgeDays",
		},
		{
			name:  "power unit range inverted",
			query: SourcingQuery{MinPowerUnits: new(50), MaxPowerUnits: new(5)},
			field: "maxPowerUnits",
		},
		{name: "unknown sort", query: SourcingQuery{Sort: "Newest"}, field: "sort"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateSourcingQuery(&tt.query)
			if tt.field == "" {
				assert.NoError(t, err)
				return
			}
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, tt.field, multiErr.Errors[0].Field)
		})
	}
}
