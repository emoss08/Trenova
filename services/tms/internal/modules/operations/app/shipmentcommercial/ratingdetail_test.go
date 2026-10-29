package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRatingDetailFromQuote_KeepsTheTariffsChoices(t *testing.T) {
	t.Parallel()

	chosen := "Flat"
	flat := decimal.NewFromInt(900)
	choices := []tarifftypes.Choice{{
		Label:    "Linehaul",
		Kind:     tarifftypes.ChoiceBest,
		Currency: "USD",
		Chosen:   &chosen,
		Options: []tarifftypes.ChoiceOption{
			{Label: "Flat", Status: tarifftypes.OptionChosen, Subtotal: &flat},
			{Label: "Volume", Status: tarifftypes.OptionRefused, Refusal: "Too light"},
		},
	}}

	detail := ratingDetailFromQuote(&ratequoteports.RatedShipment{
		Outcome: ratequote.OutcomeRated,
		Choices: choices,
		Quote:   &ratequote.RateQuote{RatedAt: 1_700_000_000},
	})

	require.NotNil(t, detail)
	assert.Equal(t, choices, detail.Choices)
	assert.EqualValues(t, 1_700_000_000, detail.RatedAt)
}

func TestRatingDetailFromQuote_KeepsTheTraceStatementsAndLookupsTogether(t *testing.T) {
	t.Parallel()

	statements := rules.RawJSON(`[{"kind":"charge","label":"Flat Rate","display":"$2450.00"}]`)
	lookups := rules.RawJSON(`[{"table":"fuel","row":3}]`)
	detail := ratingDetailFromQuote(&ratequoteports.RatedShipment{
		Outcome: ratequote.OutcomeRated,
		Quote: &ratequote.RateQuote{
			RatedAt: 1_700_000_000,
			Trace:   &ratetypes.Trace{Statements: statements, Lookups: lookups},
		},
	})

	require.NotNil(t, detail.Trace)
	assert.Equal(t, &shipment.RatingTrace{Statements: statements, Lookups: lookups}, detail.Trace)
}

func TestRatingDetailFromQuote_LeavesTheTraceOutWhenNoStatementRan(t *testing.T) {
	t.Parallel()

	detail := ratingDetailFromQuote(&ratequoteports.RatedShipment{
		Outcome: ratequote.OutcomeRated,
		Quote:   &ratequote.RateQuote{RatedAt: 1_700_000_000, Trace: &ratetypes.Trace{}},
	})

	assert.Nil(t, detail.Trace)
}

func reviewedRating(reviews ...string) *ratequoteports.RatedShipment {
	bill := &tarifftypes.Bill{Currency: "USD", Total: decimal.NewFromInt(100)}
	for _, text := range reviews {
		bill.Reviews = append(bill.Reviews, tarifftypes.Note{Text: text})
	}
	return &ratequoteports.RatedShipment{
		Outcome: ratequote.OutcomeRated,
		Bill:    bill,
		Quote:   &ratequote.RateQuote{RatedAt: 1_800_000_000},
	}
}

func approvedDetail(reviews ...string) *shipment.RatingDetail {
	detail := ratingDetailFromQuote(reviewedRating(reviews...))
	changed, err := shipmentrating.ApproveReview(detail, pulid.MustNew("usr_"), 1_790_000_000)
	if err != nil || !changed {
		panic("the fixture rating must be approvable")
	}
	return detail
}

func TestRatingDetail_HoldsANewlyReviewedRating(t *testing.T) {
	t.Parallel()

	calc := &Calculator{now: func() int64 { return 1 }}
	entity := &shipment.Shipment{}

	detail := calc.ratingDetail(entity, reviewedRating("Liftgate on a dock"), &ratingPass{})

	assert.Equal(t, []string{"Liftgate on a dock"}, shipmentrating.ReviewReasons(detail))
	assert.True(t, shipmentrating.DetailAwaitsReview(detail))
	assert.Nil(t, detail.ReviewApproval)
}

func TestRatingDetail_KeepsTheApprovalWhileTheReviewsAreTheSame(t *testing.T) {
	t.Parallel()

	calc := &Calculator{now: func() int64 { return 1 }}
	previous := approvedDetail("Holiday pickup", "Liftgate on a dock")
	entity := &shipment.Shipment{RatingDetail: previous}

	detail := calc.ratingDetail(
		entity,
		reviewedRating("Liftgate on a dock", "Holiday pickup"),
		&ratingPass{},
	)

	require.NotNil(t, detail.ReviewApproval)
	assert.Equal(t, previous.ReviewApproval.ApprovedByID, detail.ReviewApproval.ApprovedByID)
	assert.False(t, shipmentrating.DetailAwaitsReview(detail))
}

func TestRatingDetail_HoldsAgainWhenTheReviewsChange(t *testing.T) {
	t.Parallel()

	calc := &Calculator{now: func() int64 { return 1 }}
	entity := &shipment.Shipment{RatingDetail: approvedDetail("Holiday pickup")}

	detail := calc.ratingDetail(
		entity,
		reviewedRating("Holiday pickup", "Weight over 40,000 lb"),
		&ratingPass{},
	)

	assert.Nil(t, detail.ReviewApproval)
	assert.True(t, shipmentrating.DetailAwaitsReview(detail))
}

func TestRatingDetail_ClearsTheHoldWhenARerateHasNoReviews(t *testing.T) {
	t.Parallel()

	calc := &Calculator{now: func() int64 { return 1 }}
	held := ratingDetailFromQuote(reviewedRating("Holiday pickup"))
	entity := &shipment.Shipment{RatingDetail: held}

	detail := calc.ratingDetail(entity, reviewedRating(), &ratingPass{})

	assert.Empty(t, shipmentrating.ReviewReasons(detail))
	assert.Nil(t, detail.ReviewApproval)
	assert.False(t, shipmentrating.DetailAwaitsReview(detail))
}

func TestApplyQuote_CarriesTheApprovalOfTheSameReviews(t *testing.T) {
	t.Parallel()

	calc := &Calculator{now: func() int64 { return 1 }}
	entity := &shipment.Shipment{RatingDetail: approvedDetail("Holiday pickup")}

	calc.applyQuote(entity, reviewedRating("Holiday pickup"))
	assert.False(t, shipmentrating.AwaitsReview(entity))

	calc.applyQuote(entity, reviewedRating("Detention over 4 hours"))
	assert.True(t, shipmentrating.AwaitsReview(entity))
}

func TestRatingDetail_KeepsALockedRatesApproval(t *testing.T) {
	t.Parallel()

	calc := &Calculator{now: func() int64 { return 1 }}
	previous := approvedDetail("Holiday pickup")
	entity := &shipment.Shipment{RateLocked: true, RatingDetail: previous}

	detail := calc.ratingDetail(entity, reviewedRating("Other"), &ratingPass{})

	assert.Same(t, previous, detail)
	assert.False(t, shipmentrating.DetailAwaitsReview(detail))
}
