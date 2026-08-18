package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateCoverageAdvisory(t *testing.T) {
	t.Parallel()

	agreementID := pulid.MustNew("ragr_")

	tests := []struct {
		name     string
		entity   *shipment.Shipment
		wantNil  bool
		contains string
	}{
		{
			name: "a shipment the contract priced is fine",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeRated),
				},
			},
			wantNil: true,
		},
		{
			name: "a shipment its formula template priced is fine",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeFormulaFallback),
				},
			},
			wantNil: true,
		},
		{
			name: "a hand-set rate needs no contract",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeManualOverride),
				},
			},
			wantNil: true,
		},
		{
			name: "nothing covered the lane and nothing else could price it",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeNoRateFound),
				},
			},
			contains: "no rating method is set",
		},
		{
			name: "nothing covered the lane but a template did exist",
			entity: &shipment.Shipment{
				FormulaTemplateID: pulid.MustNew("ft_"),
				RatingDetail: &shipment.RatingDetail{
					Source: string(ratequote.OutcomeNoRateFound),
				},
			},
			contains: "No rate agreement covers this lane",
		},
		{
			name: "a rate that blew up is surfaced with its reason",
			entity: &shipment.Shipment{
				RatingDetail: &shipment.RatingDetail{
					Source:      string(ratequote.OutcomeError),
					Explanation: "division by zero",
				},
			},
			contains: "division by zero",
		},
		{
			name:     "an unrated shipment with no way to be priced",
			entity:   &shipment.Shipment{},
			contains: "cannot be priced",
		},
		{
			name:    "an unrated shipment carrying a template is left alone",
			entity:  &shipment.Shipment{FormulaTemplateID: pulid.MustNew("ft_")},
			wantNil: true,
		},
		{
			name:    "an unrated shipment carrying a contract is left alone",
			entity:  &shipment.Shipment{RateAgreementID: &agreementID},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			advisory := rateCoverageAdvisory(tt.entity)

			if tt.wantNil {
				assert.Nil(t, advisory)
				return
			}

			require.NotNil(t, advisory)
			assert.Contains(t, advisory.Message, tt.contains)
			assert.Equal(t, errortypes.SeverityRequireReview, advisory.Severity)
			assert.Equal(t, rateCoverageRuleKey, advisory.RuleKey)
			assert.Equal(t, "freightChargeAmount", advisory.Field)
		})
	}
}
