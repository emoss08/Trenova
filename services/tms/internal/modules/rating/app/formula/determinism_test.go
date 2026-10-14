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

func TestService_ValidateExpression_RefusesNonDeterministicBuiltins(t *testing.T) {
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

			err := svc.ValidateExpression(t.Context(), tc.expression, "shipment")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not available in formulas")
			assert.Contains(t, err.Error(), tc.names)
		})
	}
}

func TestService_DatesParseInUTC(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	resp, err := svc.EvaluateExpression(t.Context(), &formula.EvaluateExpressionRequest{
		Expression: `date("2025-03-01").Location().String() == "UTC" ? 1 : 0`,
		Entity:     shippedOnMarchFirst(),
		SchemaID:   "shipment",
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(1).Equal(resp.Amount),
		"date() must not depend on the server's local zone")
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

	warnings, err := svc.UnguardedNullableFields(
		t.Context(),
		`pickupDate.Hour() > 12 ? 10 : 0`,
		"shipment",
		nil,
	)
	require.NoError(t, err)
	require.NotEmpty(t, warnings)
	assert.Equal(t, "pickupDate", warnings[0].Field)
	assert.Equal(t, "coalesce(pickupDate, ratingTime)", warnings[0].Suggestion)
}
