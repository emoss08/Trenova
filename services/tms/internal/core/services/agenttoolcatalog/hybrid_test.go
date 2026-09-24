package agenttoolcatalog

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var parityQueries = []string{
	"",
	"which drivers are available",
	"trucks out of service",
	"medical cards expiring",
	"zzzz no such thing zzzz",
	"list",
	"run the report",
}

func TestRankHybrid_IsTheKeywordRankingWithoutAVector(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	semantics := []*Semantic{nil, NewSemantic(nil), {Floor: DefaultSimilarityFloor}}
	for _, query := range parityQueries {
		for _, semantic := range semantics {
			assert.Equal(t,
				catalog.Rank(nil, query, 4),
				catalog.RankHybrid(Query{Text: query, Limit: 4, Semantic: semantic}),
				"query %q", query,
			)
			assert.Equal(t,
				catalog.Rank([]string{"list_workers", "assign_move"}, query, 4),
				catalog.RankHybrid(Query{
					Allowed:  []string{"list_workers", "assign_move"},
					Text:     query,
					Limit:    4,
					Semantic: semantic,
				}),
				"query %q", query,
			)
		}
	}
}

func TestFindHybrid_IsTheKeywordSearchWithoutAVector(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	for _, query := range parityQueries {
		assert.Equal(t,
			catalog.Find(nil, query, 6),
			catalog.FindHybrid(Query{Text: query, Limit: 6}),
			"query %q", query,
		)
		assert.Equal(t,
			catalog.Find([]string{}, query, 6),
			catalog.FindHybrid(Query{Allowed: []string{}, Text: query, Limit: 6}),
			"query %q", query,
		)
	}
}

func TestFindHybrid_BreaksAFusedTieOnCatalogPosition(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	keyword := names(catalog.Find(nil, "medical cards", 6))
	require.Equal(t, []string{"list_expiring_credentials"}, keyword)

	found := catalog.FindHybrid(Query{
		Text:     "medical cards",
		Limit:    6,
		Semantic: NewSemantic(map[string]float64{"list_time_off": 0.91}),
	})
	assert.Equal(t, []string{"list_expiring_credentials", "list_time_off"}, names(found),
		"each leads one list, so the one earlier in the catalog goes first")

	found = catalog.FindHybrid(Query{
		Text:     "medical cards",
		Limit:    6,
		Semantic: NewSemantic(map[string]float64{"assign_move": 0.91}),
	})
	assert.Equal(t, []string{"assign_move", "list_expiring_credentials"}, names(found))
}

func TestFindHybrid_RanksWhatBothListsFoundAboveEitherAlone(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	found := catalog.FindHybrid(Query{
		Text:  "medical cards",
		Limit: 6,
		Semantic: NewSemantic(map[string]float64{
			"list_time_off":             0.92,
			"list_expiring_credentials": 0.9,
			"list_workers":              0.7,
		}),
	})

	assert.Equal(t,
		[]string{"list_expiring_credentials", "list_time_off", "list_workers"},
		names(found),
	)
}

func TestFindHybrid_FusesWithReciprocalRank(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	keyword := names(catalog.Find(nil, "tractor status", 6))
	require.GreaterOrEqual(t, len(keyword), 2)

	similarity := map[string]float64{
		keyword[len(keyword)-1]: 0.95,
		"list_time_off":         0.9,
	}
	found := names(catalog.FindHybrid(Query{
		Text:     "tractor status",
		Limit:    6,
		Semantic: NewSemantic(similarity),
	}))

	last := keyword[len(keyword)-1]
	lastScore := 1.0/float64(RRFK+len(keyword)) + 1.0/float64(RRFK+1)
	firstScore := 1.0 / float64(RRFK+1)
	require.Greater(t, lastScore, firstScore)
	assert.Equal(t, last, found[0],
		"the keyword leg's last hit leads once the vector ranks it first")
	assert.Contains(t, found, "list_time_off")
}

func TestFindHybrid_DropsMeaningOnlyHitsBelowTheFloor(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	found := catalog.FindHybrid(Query{
		Text:  "zzzz qqqq",
		Limit: 6,
		Semantic: NewSemantic(map[string]float64{
			"list_time_off": DefaultSimilarityFloor - 0.01,
			"assign_move":   0.2,
		}),
	})
	assert.Empty(t, found, "nonsense still finds nothing")

	found = catalog.FindHybrid(Query{
		Text:  "zzzz qqqq",
		Limit: 6,
		Semantic: NewSemantic(map[string]float64{
			"list_time_off": DefaultSimilarityFloor,
			"assign_move":   0.2,
		}),
	})
	assert.Equal(t, []string{"list_time_off"}, names(found))
}

func TestFindHybrid_KeepsAKeywordHitWhateverItsSimilarity(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	found := catalog.FindHybrid(Query{
		Text:     "medical cards",
		Limit:    6,
		Semantic: NewSemantic(map[string]float64{"list_expiring_credentials": 0.01}),
	})

	assert.Equal(t, []string{"list_expiring_credentials"}, names(found))
}

func TestFindHybrid_SkipsWhatTheTurnAlreadyCarriesOnTheVectorSide(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	found := catalog.FindHybrid(Query{
		Text:  "zzzz",
		Limit: 6,
		Semantic: NewSemantic(map[string]float64{
			"list_time_off": 0.9,
			"list_workers":  0.8,
		}),
		SkipSemantic: map[string]struct{}{"list_time_off": {}},
	})

	assert.Equal(t, []string{"list_workers"}, names(found))
}

func TestRankHybrid_FillsItsSlotsAndNeverWidens(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	allowed := []string{"list_workers", "list_tractors", "assign_move", "run_report"}
	ranked := catalog.RankHybrid(Query{
		Allowed: allowed,
		Text:    "zzzz",
		Limit:   3,
		Semantic: NewSemantic(map[string]float64{
			"list_time_off": 0.99,
			"run_report":    0.8,
		}),
	})

	require.Len(t, ranked, 3, "preselection always fills its slots")
	assert.Equal(t, "run_report", ranked[0].Name)
	assert.NotContains(t, names(ranked), "list_time_off",
		"a tool the agent does not hold is never offered however close it is")
	for _, name := range names(ranked) {
		assert.Contains(t, allowed, name)
	}
}

func TestItems_KeyEachToolByTheHashOfItsText(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()
	items := catalog.Items()
	require.Len(t, items, len(catalog.Names()))

	for idx, item := range items {
		descriptor, ok := catalog.Descriptor(item.Key)
		require.True(t, ok)
		assert.Equal(t, catalog.Names()[idx], item.Key)
		assert.Equal(t, DescriptorText(descriptor), item.Text)
		assert.Equal(t, hashutils.SHA256Hex(item.Text), item.ContentHash)
	}

	items[0].Key = "changed"
	assert.NotEqual(t, "changed", catalog.Items()[0].Key, "callers get a copy")

	edited := New([]serviceports.AgentToolDescriptor{
		descriptor("list_workers", "List every worker."),
	})
	original := ""
	for _, item := range catalog.Items() {
		if item.Key == "list_workers" {
			original = item.ContentHash
		}
	}
	require.NotEmpty(t, original)
	assert.NotEqual(t, original, edited.Items()[0].ContentHash,
		"an edited description is a new hash, so it is embedded again")
}

func TestDescriptorText_NamesTheToolAndWhatPeopleCallIt(t *testing.T) {
	t.Parallel()

	text := DescriptorText(serviceports.AgentToolDescriptor{
		Name:        "get_my_home_layout",
		Description: " Read the person's home page. ",
		SearchTerms: []string{"my dashboard", "home"},
	})

	assert.Equal(t,
		"get my home layout: Read the person's home page.\nAlso asked for as: my dashboard, home",
		text,
	)
}
