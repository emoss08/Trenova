package detentionsimulation

import (
	"sort"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// BacktestRequest re-prices settled history under proposed terms.
type BacktestRequest struct {
	Policy        *detentionpolicy.DetentionPolicy `json:"policy"`
	CompareToID   *pulid.ID                        `json:"compareToPolicyId"`
	From          int64                            `json:"from"`
	To            int64                            `json:"to"`
	Limit         int                              `json:"limit"`
	DriverPayRate *string                          `json:"driverPayRate"`

	// AssumeNoticeCompliance models the notice as having been sent on time.
	// Defaults to true, because a backtest is asking what the *terms* would
	// earn, not re-litigating past process failures. Set false to see what the
	// same history would have produced given the notices actually sent.
	AssumeNoticeCompliance *bool `json:"assumeNoticeCompliance"`
}

// BacktestBucket is one grouped slice of the backtest result.
type BacktestBucket struct {
	Key             string              `json:"key"`
	Label           string              `json:"label"`
	StopCount       int                 `json:"stopCount"`
	BillableCount   int                 `json:"billableCount"`
	ProposedAmount  decimal.Decimal     `json:"proposedAmount"`
	BaselineAmount  decimal.Decimal     `json:"baselineAmount"`
	Delta           decimal.Decimal     `json:"delta"`
	DriverPayAmount decimal.NullDecimal `json:"driverPayAmount"`
	NetMargin       decimal.NullDecimal `json:"netMargin"`
}

// BacktestResult is the projected impact of a policy change.
type BacktestResult struct {
	StopsEvaluated      int                 `json:"stopsEvaluated"`
	StopsMatched        int                 `json:"stopsMatched"`
	StopsBillable       int                 `json:"stopsBillable"`
	StopsForfeited      int                 `json:"stopsForfeited"`
	StopsSuppressed     int                 `json:"stopsSuppressed"`
	ProposedRevenue     decimal.Decimal     `json:"proposedRevenue"`
	BaselineRevenue     decimal.Decimal     `json:"baselineRevenue"`
	RevenueDelta        decimal.Decimal     `json:"revenueDelta"`
	ProposedDriverPay   decimal.NullDecimal `json:"proposedDriverPay"`
	ProposedNetMargin   decimal.NullDecimal `json:"proposedNetMargin"`
	NegativeMarginStops *int                `json:"negativeMarginStops"`
	ByCustomer          []BacktestBucket    `json:"byCustomer"`
	ByFacility          []BacktestBucket    `json:"byFacility"`
	From                int64               `json:"from"`
	To                  int64               `json:"to"`
	Truncated           bool                `json:"truncated"`
}

const defaultBacktestLimit = 2000

func accumulate(
	buckets map[string]*BacktestBucket,
	key string,
	computed detentionengine.ComputeResult,
	baseline decimal.Decimal,
) {
	bucket, ok := buckets[key]
	if !ok {
		bucket = &BacktestBucket{
			Key:             key,
			ProposedAmount:  decimal.Zero,
			BaselineAmount:  decimal.Zero,
			Delta:           decimal.Zero,
			DriverPayAmount: decimal.NewNullDecimal(decimal.Zero),
			NetMargin:       decimal.NewNullDecimal(decimal.Zero),
		}
		buckets[key] = bucket
	}

	bucket.StopCount++
	if computed.BillableAmount.IsPositive() {
		bucket.BillableCount++
	}

	bucket.ProposedAmount = bucket.ProposedAmount.Add(computed.BillableAmount)
	bucket.BaselineAmount = bucket.BaselineAmount.Add(baseline)
	bucket.Delta = bucket.ProposedAmount.Sub(bucket.BaselineAmount)
	bucket.DriverPayAmount = addNull(bucket.DriverPayAmount, computed.DriverPayAmount)
	bucket.NetMargin = addNull(bucket.NetMargin, computed.NetMargin)
}

func addNull(total decimal.NullDecimal, amount decimal.Decimal) decimal.NullDecimal {
	return decimal.NewNullDecimal(total.Decimal.Add(amount))
}

// rankBuckets orders by the magnitude of the change, so the biggest movers in
// either direction surface first.
func rankBuckets(buckets map[string]*BacktestBucket) []BacktestBucket {
	ranked := make([]BacktestBucket, 0, len(buckets))
	for _, bucket := range buckets {
		ranked = append(ranked, *bucket)
	}

	sort.SliceStable(ranked, func(a, b int) bool {
		return ranked[a].Delta.Abs().GreaterThan(ranked[b].Delta.Abs())
	})

	return ranked
}

// noticeSentFor models notice compliance for a replayed stop.
func noticeSentFor(assumeCompliance bool, arrival *int64) *int64 {
	if !assumeCompliance {
		return nil
	}
	return arrival
}

func appointmentStartFromFact(fact *detentionports.StopFact) *int64 {
	if fact.ScheduledWindowStart <= 0 {
		return nil
	}
	start := fact.ScheduledWindowStart
	return &start
}

func derefOr(value *int64, fallback int64) int64 {
	if value == nil {
		return fallback
	}
	return *value
}
