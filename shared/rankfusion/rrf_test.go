package rankfusion_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/rankfusion"
	"github.com/stretchr/testify/assert"
)

func TestReciprocalSumsEachListsRank(t *testing.T) {
	t.Parallel()

	scores := rankfusion.Reciprocal(60,
		[]string{"a", "b", "c"},
		[]string{"c", "a"},
	)

	assert.InDelta(t, 1.0/61+1.0/62, scores["a"], 1e-12)
	assert.InDelta(t, 1.0/62, scores["b"], 1e-12)
	assert.InDelta(t, 1.0/63+1.0/61, scores["c"], 1e-12)
	assert.Greater(t, scores["a"], scores["b"])
}

func TestReciprocalCountsADuplicateOnceAtItsBestRank(t *testing.T) {
	t.Parallel()

	scores := rankfusion.Reciprocal(0, []string{"a", "a", "b"})

	assert.InDelta(t, 1.0/61, scores["a"], 1e-12)
	assert.InDelta(t, 1.0/62, scores["b"], 1e-12)
}

func TestReciprocalOfNoListsIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, rankfusion.Reciprocal[string](60))
}

func TestReciprocalTiersGivesATiedTierItsFirstRank(t *testing.T) {
	t.Parallel()

	scores := rankfusion.ReciprocalTiers(60,
		[][]string{{"a", "b"}, {"c"}},
		[][]string{{"c"}, {"a"}},
	)

	assert.InDelta(t, 1.0/61+1.0/62, scores["a"], 1e-12)
	assert.InDelta(t, 1.0/61, scores["b"], 1e-12)
	assert.InDelta(t, 1.0/63+1.0/61, scores["c"], 1e-12)
}

func TestReciprocalTiersMatchesReciprocalWhenNothingTies(t *testing.T) {
	t.Parallel()

	plain := rankfusion.Reciprocal(60, []string{"a", "b", "c"}, []string{"c", "a"})
	tiered := rankfusion.ReciprocalTiers(60,
		[][]string{{"a"}, {"b"}, {"c"}},
		[][]string{{"c"}, {"a"}},
	)

	assert.Len(t, tiered, len(plain))
	for key, score := range plain {
		assert.InDelta(t, score, tiered[key], 1e-12, key)
	}
}

func TestReciprocalTiersCountsADuplicateOnceAtItsBestRank(t *testing.T) {
	t.Parallel()

	scores := rankfusion.ReciprocalTiers(0, [][]string{{"a"}, {"a", "b"}})

	assert.InDelta(t, 1.0/61, scores["a"], 1e-12)
	assert.InDelta(t, 1.0/62, scores["b"], 1e-12)
}
