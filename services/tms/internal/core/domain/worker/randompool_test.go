package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomPool_TargetsFor(t *testing.T) {
	t.Parallel()

	pool := &worker.DOTRandomPool{
		Period:             worker.RandomPeriodQuarterly,
		DrugRatePercent:    worker.DOTMinimumDrugRatePercent,
		AlcoholRatePercent: worker.DOTMinimumAlcoholRatePercent,
	}

	// 100 drivers at 50% a year is 50 collections, spread over four rounds.
	// 12.5 rounds up: taking 12 every quarter would finish the year at 48% and
	// under the minimum.
	drug, alcohol := pool.TargetsFor(100)
	assert.Equal(t, 13, drug)
	assert.Equal(t, 3, alcohol)

	// A pool of one still owes a draw: rounding down would mean a small carrier
	// never tests anybody.
	drug, alcohol = pool.TargetsFor(1)
	assert.Equal(t, 1, drug)
	assert.Equal(t, 1, alcohol)

	drug, alcohol = pool.TargetsFor(0)
	assert.Equal(t, 0, drug)
	assert.Equal(t, 0, alcohol)

	annual := &worker.DOTRandomPool{
		Period:             worker.RandomPeriodAnnual,
		DrugRatePercent:    50,
		AlcoholRatePercent: 10,
	}
	drug, alcohol = annual.TargetsFor(100)
	assert.Equal(t, 50, drug)
	assert.Equal(t, 10, alcohol)

	// A target can never exceed the pool, however the rates are set.
	huge := &worker.DOTRandomPool{
		Period:             worker.RandomPeriodMonthly,
		DrugRatePercent:    100,
		AlcoholRatePercent: 100,
	}
	drug, _ = huge.TargetsFor(3)
	assert.LessOrEqual(t, drug, 3)
}

func TestRandomPool_MeetsDOTMinimums(t *testing.T) {
	t.Parallel()

	assert.True(t, (&worker.DOTRandomPool{DrugRatePercent: 50, AlcoholRatePercent: 10}).
		MeetsDOTMinimums())
	assert.False(t, (&worker.DOTRandomPool{DrugRatePercent: 25, AlcoholRatePercent: 10}).
		MeetsDOTMinimums())
	assert.False(t, (&worker.DOTRandomPool{DrugRatePercent: 50, AlcoholRatePercent: 5}).
		MeetsDOTMinimums())
}

func TestRandomPool_Includes(t *testing.T) {
	t.Parallel()

	all := &worker.DOTRandomPool{}
	assert.True(t, all.Includes("Employee"))

	some := &worker.DOTRandomPool{IncludedDriverTypes: []string{"Employee"}}
	assert.True(t, some.Includes("Employee"))
	assert.False(t, some.Includes("OwnerOperator"))
}

func TestSelectRandom(t *testing.T) {
	t.Parallel()

	candidates := make([]pulid.ID, 0, 50)
	for range 50 {
		candidates = append(candidates, pulid.MustNew("wrk_"))
	}

	first := worker.SelectRandom(candidates, "seed-one", 10)
	require.Len(t, first, 10)

	// The whole point of recording the seed is that the draw can be re-run: the
	// same seed over the same roster must produce the same names in the same
	// order, years later.
	again := worker.SelectRandom(candidates, "seed-one", 10)
	assert.Equal(t, first, again)

	other := worker.SelectRandom(candidates, "seed-two", 10)
	assert.NotEqual(t, first, other)

	// Nobody is drawn twice in one round.
	seen := make(map[pulid.ID]bool, len(first))
	for _, id := range first {
		assert.False(t, seen[id], "%s drawn twice", id)
		seen[id] = true
	}

	// Asking for more names than the pool holds returns the pool, not an error
	// and not a short slice padded with anything.
	assert.Len(t, worker.SelectRandom(candidates, "seed-one", 500), 50)
	assert.Nil(t, worker.SelectRandom(candidates, "seed-one", 0))
	assert.Nil(t, worker.SelectRandom(nil, "seed-one", 5))
}

// The roster arrives in whatever order the database returned it. A draw that
// depended on that order would not be random, and two runs of the same round
// could disagree.
func TestSelectRandom_IgnoresInputOrder(t *testing.T) {
	t.Parallel()

	candidates := make([]pulid.ID, 0, 20)
	for range 20 {
		candidates = append(candidates, pulid.MustNew("wrk_"))
	}

	reversed := make([]pulid.ID, len(candidates))
	for i, id := range candidates {
		reversed[len(candidates)-1-i] = id
	}

	assert.Equal(t,
		worker.SelectRandom(candidates, "seed", 7),
		worker.SelectRandom(reversed, "seed", 7),
	)
}

func TestPeriodKeyFor(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)

	assert.Equal(t, "2026-M08", worker.PeriodKeyFor(worker.RandomPeriodMonthly, at))
	assert.Equal(t, "2026-Q3", worker.PeriodKeyFor(worker.RandomPeriodQuarterly, at))
	assert.Equal(t, "2026-H2", worker.PeriodKeyFor(worker.RandomPeriodSemiAnnual, at))
	assert.Equal(t, "2026", worker.PeriodKeyFor(worker.RandomPeriodAnnual, at))
}

func TestPeriodBoundsFor(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)

	start, end := worker.PeriodBoundsFor(worker.RandomPeriodQuarterly, at)
	assert.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC).Unix(), start)
	assert.Equal(t, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC).Unix(), end)

	// A December round must roll into the next year rather than wrapping to
	// January of the same one.
	december := time.Date(2026, time.December, 31, 23, 0, 0, 0, time.UTC)
	start, end = worker.PeriodBoundsFor(worker.RandomPeriodMonthly, december)
	assert.Equal(t, time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC).Unix(), start)
	assert.Equal(t, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), end)
}

func TestRandomDraw_ShortOfTarget(t *testing.T) {
	t.Parallel()

	draw := &worker.DOTRandomDraw{
		DrugTarget:      13,
		DrugSelected:    9,
		AlcoholTarget:   3,
		AlcoholSelected: 3,
	}

	drug, alcohol := draw.ShortOfTarget()
	assert.Equal(t, int32(4), drug)
	assert.Equal(t, int32(0), alcohol)
}
