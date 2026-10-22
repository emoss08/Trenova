package formula_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Check_RefusesNonDeterministicBuiltins(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	cases := map[string]struct {
		expression string
		names      string
	}{
		"now":      {`now().Year() == 2026 ? 10 : 0`, "ratingTime"},
		"toJSON":   {`len(toJSON(stops)) > 2 ? 10 : 0`, "toJSON"},
		"fromJSON": {`fromJSON("1") + 1`, "fromJSON"},
		"repeat":   {`len(repeat("x", 3))`, "repeat"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			check := checkOne(t, svc, tc.expression)
			refusal := check.FirstError()
			require.NotNil(t, refusal, tc.expression)
			assert.Equal(t, "disabled-function", refusal.Code)
			assert.Contains(t, refusal.Message.String(), tc.names)
		})
	}
}

func shippedOnMarchFirst() *shipment.Shipment {
	shipped := time.Date(2025, time.March, 1, 15, 0, 0, 0, time.UTC).Unix()
	return &shipment.Shipment{
		ServiceType:    &servicetype.ServiceType{Code: "STD"},
		ActualShipDate: &shipped,
		CreatedAt:      time.Date(2025, time.February, 20, 9, 0, 0, 0, time.UTC).Unix(),
	}
}

func TestService_Rate_RatingTimeIsTheShipmentsRatingDate(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: activeTemplate(
			`ratingTime.Year() == 2025 && ratingTime.Day() == 1 ? 10 : 0`,
		),
		Entity: shippedOnMarchFirst(),
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(10).Equal(resp.Amount), resp.Amount.String())

	var recorded *formulatypes.VariableProvenance
	for i := range resp.Receipt.Variables {
		if resp.Receipt.Variables[i].Name == "ratingTime" {
			recorded = &resp.Receipt.Variables[i]
		}
	}
	require.NotNil(t, recorded, "the receipt records the rating time it priced at")
	assert.Equal(t, formulatypes.ValueSourceProvided, recorded.Source)

	unshipped := shippedOnMarchFirst()
	unshipped.ActualShipDate = nil
	resp, err = svc.Rate(t.Context(), &formula.RateRequest{
		Template: activeTemplate(`ratingTime.Day()`),
		Entity:   unshipped,
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(20).Equal(resp.Amount),
		"without a ship date the shipment rates as of when it was entered")
}

func TestService_Calculate_RatingTimeFollowsTheRequestedRatingDate(t *testing.T) {
	t.Parallel()

	template := activeTemplate(`ratingTime.Year() == 2024 ? 10 : 0`)
	svc := setupServiceWithRepo(t, &singleTemplateRepo{template: template})

	resp, err := svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
		TemplateID: template.ID,
		Entity:     shippedOnMarchFirst(),
		RatingDate: time.Date(2024, time.July, 4, 0, 0, 0, 0, time.UTC).Unix(),
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(10).Equal(resp.Amount), resp.Amount.String())
}

func TestService_NullableDateGuardSuggestsRatingTime(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	check := checkOne(t, svc, `pickupDate.Hour() > 12 ? 10 : 0`)
	assert.Equal(t, []string{"pickupDate"}, check.Unguarded())
	require.NotEmpty(t, check.Diagnostics[0].Fixes)
	assert.Equal(
		t,
		"coalesce(pickupDate, ratingTime)",
		check.Diagnostics[0].Fixes[0].Changes[0].Insert,
	)
}
