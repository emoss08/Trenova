package agentevalgate_test

import (
	"errors"
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoreRetrieval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		relevant []string
		ranked   []string
		first    int
		recall   float64
		ndcg     float64
	}{
		{
			name:     "the only relevant item first",
			relevant: []string{"a"},
			ranked:   []string{"a", "b", "c"},
			first:    1,
			recall:   1,
			ndcg:     1,
		},
		{
			name:     "the only relevant item third",
			relevant: []string{"a"},
			ranked:   []string{"b", "c", "a"},
			first:    3,
			recall:   1,
			ndcg:     1 / math.Log2(4),
		},
		{
			name:     "one of two relevant items in the top five",
			relevant: []string{"a", "z"},
			ranked:   []string{"a", "b", "c", "d", "e", "z"},
			first:    1,
			recall:   0.5,
			ndcg:     (1 + 1/math.Log2(7)) / (1 + 1/math.Log2(3)),
		},
		{
			name:     "nothing relevant found",
			relevant: []string{"a"},
			ranked:   []string{"b", "c"},
		},
		{
			name:     "a repeated key counts once",
			relevant: []string{"a", "b"},
			ranked:   []string{"a", "a", "b"},
			first:    1,
			recall:   1,
			ndcg:     (1 + 1/math.Log2(4)) / (1 + 1/math.Log2(3)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outcome := agentevalgate.ScoreRetrieval(
				agentevalgate.RetrievalCase{Query: "q", Relevant: tt.relevant},
				tt.ranked,
			)
			assert.Equal(t, tt.first, outcome.FirstHit)
			assert.InDelta(t, tt.recall, outcome.RecallAt5, 1e-9)
			assert.InDelta(t, tt.ndcg, outcome.NDCGAt10, 1e-9)
		})
	}
}

func TestEvaluateRetrievalAveragesEveryCase(t *testing.T) {
	t.Parallel()

	suite := &agentevalgate.RetrievalSuite{
		Corpus: []agentevalgate.RetrievalItem{{Key: "a"}, {Key: "b"}},
		Cases: []agentevalgate.RetrievalCase{
			{Query: "first", Relevant: []string{"a"}},
			{Query: "second", Relevant: []string{"b"}},
		},
	}
	require.NoError(t, suite.Validate())

	report, err := agentevalgate.EvaluateRetrieval(suite, func(query string) ([]string, error) {
		if query == "first" {
			return []string{"a", "b"}, nil
		}
		return []string{"a", "b"}, nil
	})
	require.NoError(t, err)
	assert.InDelta(t, 1.0, report.RecallAt5, 1e-9)
	assert.InDelta(t, 0.75, report.MRR, 1e-9)
	assert.Equal(t, agentevalgate.RetrievalFloors{
		Cases:     2,
		RecallAt5: 1,
		MRR:       0.75,
		NDCGAt10:  0.81,
	}, report.Floors())
	assert.Contains(t, report.Misses(), `"second" wants b, first at 2`)
	assert.Empty(t, report.Below(report.Floors()))
	assert.Len(t, report.Below(agentevalgate.RetrievalFloors{RecallAt5: 1.1, MRR: 1, NDCGAt10: 1}), 3)
}

func TestEvaluateRetrievalReportsARetrieverError(t *testing.T) {
	t.Parallel()

	suite := &agentevalgate.RetrievalSuite{
		Corpus: []agentevalgate.RetrievalItem{{Key: "a"}},
		Cases:  []agentevalgate.RetrievalCase{{Query: "q", Relevant: []string{"a"}}},
	}
	_, err := agentevalgate.EvaluateRetrieval(suite, func(string) ([]string, error) {
		return nil, errors.New("database is down")
	})
	require.ErrorContains(t, err, "database is down")
}

func TestRetrievalSuiteValidate(t *testing.T) {
	t.Parallel()

	duplicate := &agentevalgate.RetrievalSuite{
		Corpus: []agentevalgate.RetrievalItem{{Key: "a"}, {Key: "a"}},
	}
	require.ErrorContains(t, duplicate.Validate(), "used twice")

	unknown := &agentevalgate.RetrievalSuite{
		Corpus: []agentevalgate.RetrievalItem{{Key: "a"}},
		Cases:  []agentevalgate.RetrievalCase{{Query: "q", Relevant: []string{"b"}}},
	}
	require.ErrorContains(t, unknown.Validate(), "does not hold")
}

func TestRetrievalSuitesLoad(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"evals/documentretrieval.yaml",
		"evals/inboxretrieval.yaml",
		"evals/memoryretrieval.yaml",
	} {
		suite, err := agentevalgate.LoadRetrievalSuite(path)
		require.NoError(t, err, path)
		assert.GreaterOrEqual(t, len(suite.Cases), 12, path)
		assert.GreaterOrEqual(t, len(suite.Corpus), 12, path)
	}
}
