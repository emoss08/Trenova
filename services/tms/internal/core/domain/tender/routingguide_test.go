package tender_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func locationPtr() *pulid.ID {
	id := pulid.MustNew("loc_")
	return &id
}

func validEntry(rank int16) *tender.RoutingGuideEntry {
	return &tender.RoutingGuideEntry{
		CarrierID:       pulid.MustNew("car_"),
		Rank:            rank,
		OfferTTLSeconds: tender.DefaultOfferTTLSeconds,
	}
}

func TestRoutingGuideComputeSpecificity(t *testing.T) {
	tests := []struct {
		name  string
		guide tender.RoutingGuide
		want  int16
	}{
		{
			name: "location pair",
			guide: tender.RoutingGuide{
				OriginLocationID:      locationPtr(),
				DestinationLocationID: locationPtr(),
			},
			want: tender.SpecificityLocationPair,
		},
		{
			name: "city pair",
			guide: tender.RoutingGuide{
				OriginCity:       "Dallas",
				OriginState:      "TX",
				DestinationCity:  "Memphis",
				DestinationState: "TN",
			},
			want: tender.SpecificityCityPair,
		},
		{
			name: "state pair",
			guide: tender.RoutingGuide{
				OriginState:      "TX",
				DestinationState: "TN",
			},
			want: tender.SpecificityStatePair,
		},
		{
			name: "mixed tiers invalid",
			guide: tender.RoutingGuide{
				OriginLocationID: locationPtr(),
				DestinationState: "TN",
			},
			want: 0,
		},
		{
			name:  "empty invalid",
			guide: tender.RoutingGuide{},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.guide.ComputeSpecificity())
		})
	}
}

func TestRoutingGuideValidate(t *testing.T) {
	t.Run("valid location pair guide", func(t *testing.T) {
		guide := &tender.RoutingGuide{
			Name:                  "Dallas to Memphis",
			OriginLocationID:      locationPtr(),
			DestinationLocationID: locationPtr(),
			Entries:               []*tender.RoutingGuideEntry{validEntry(1), validEntry(2)},
		}
		multiErr := errortypes.NewMultiError()
		guide.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), "errors: %v", multiErr)
		assert.Equal(t, tender.SpecificityLocationPair, guide.Specificity)
	})

	t.Run("mixed tier rejected", func(t *testing.T) {
		guide := &tender.RoutingGuide{
			Name:             "Bad guide",
			OriginLocationID: locationPtr(),
			DestinationState: "TN",
			Entries:          []*tender.RoutingGuideEntry{validEntry(1)},
		}
		multiErr := errortypes.NewMultiError()
		guide.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("location combined with city rejected", func(t *testing.T) {
		guide := &tender.RoutingGuide{
			Name:                  "Bad guide",
			OriginLocationID:      locationPtr(),
			OriginCity:            "Dallas",
			DestinationLocationID: locationPtr(),
			Entries:               []*tender.RoutingGuideEntry{validEntry(1)},
		}
		multiErr := errortypes.NewMultiError()
		guide.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("duplicate ranks rejected", func(t *testing.T) {
		guide := &tender.RoutingGuide{
			Name:                  "Dup ranks",
			OriginLocationID:      locationPtr(),
			DestinationLocationID: locationPtr(),
			Entries:               []*tender.RoutingGuideEntry{validEntry(1), validEntry(1)},
		}
		multiErr := errortypes.NewMultiError()
		guide.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("no entries rejected", func(t *testing.T) {
		guide := &tender.RoutingGuide{
			Name:                  "Empty guide",
			OriginLocationID:      locationPtr(),
			DestinationLocationID: locationPtr(),
		}
		multiErr := errortypes.NewMultiError()
		guide.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})
}
