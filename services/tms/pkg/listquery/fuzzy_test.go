package listquery

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuzzyWords_KeepTheWordsLongEnoughToSpellClose(t *testing.T) {
	t.Parallel()

	cases := []struct {
		query string
		want  []string
	}{
		{"FreshHaul", []string{"freshhaul"}},
		{"Acme manufacuring co", []string{"acme", "manufacuring"}},
		{"St. Paul's dock", []string{"paul", "dock"}},
		{"smith Smith SMITH", []string{"smith"}},
		{"alpha bravo charlie delta echoes", []string{"alpha", "bravo", "charlie", "delta"}},
		{"store #1123", []string{"store", "1123"}},
		{"a bc def", nil},
		{"", nil},
		{`"dry van"`, nil},
		{"dry -van", nil},
		{"dry or van", nil},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			t.Parallel()

			got := fuzzyWords(tc.query)
			if len(tc.want) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCatalog_KeepsOnlyFuzzyFieldsOfStoredTextColumns(t *testing.T) {
	t.Parallel()

	table := newTrucksTable("test_fuzzy_trucks", "ft", testOwners)
	cat := listcatalog.Build(table, &domaintypes.PostgresSearchConfig{
		FuzzyFields: []domaintypes.FuzzyField{
			{"code"},
			{"code", "description"},
			{"year"},
			{"status"},
			{"search_vector"},
			{"missing"},
			{"code", "missing"},
			{},
		},
	})

	code := table.Column("code")
	description := table.Column("description")
	assert.Equal(t, [][]int32{{code}, {code, description}}, cat.Fuzzy)
}

func closeSpelling(n int) string {
	arg := "$" + strconv.Itoa(n) + "::text"

	return "(" + arg + ` <% t."code" OR ` + arg + ` <% ((t."code" || ' ') || t."description"))`
}

func TestFuzzy_EveryWordMatchesACloseSpellingAndThePhraseRanks(t *testing.T) {
	t.Parallel()

	q := testTrucks.Query()
	p := newPlan(q, nil)
	match, ok := p.fuzzy(&pagination.QueryOptions{Query: "Acme manufacuring co"})
	require.True(t, ok)

	q.Where(match.where).Order(match.score.Desc()).Limit(10)
	sql, args := render(t, q)

	assert.Contains(t, sql, truckTenant+" AND ("+closeSpelling(3)+" AND "+closeSpelling(4)+")")
	assert.Contains(t, sql, `ORDER BY greatest("word_similarity"($5::text, t."code"), `+
		`"word_similarity"($5::text, ((t."code" || ' ') || t."description"))) DESC LIMIT 10`)
	assert.Equal(t, []any{"org_1", "bu_1", "acme", "manufacuring", "acme manufacuring"}, args)
}

func TestFuzzy_OneWordRanksByItself(t *testing.T) {
	t.Parallel()

	q := testTrucks.Query()
	p := newPlan(q, nil)
	match, ok := p.fuzzy(&pagination.QueryOptions{Query: "frshhaul"})
	require.True(t, ok)

	q.Where(match.where).Order(match.score.Desc()).Limit(10)
	sql, args := render(t, q)

	assert.Contains(t, sql, truckTenant+" AND ("+closeSpelling(3)+")")
	assert.Contains(t, sql, `"word_similarity"($3::text, t."code")`)
	assert.Equal(t, []any{"org_1", "bu_1", "frshhaul"}, args)
}

func TestFuzzy_NotOfferedWithoutFieldsOrWords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		plan   *plan
		filter *pagination.QueryOptions
	}{
		{
			name:   "a model without fuzzy fields",
			plan:   newPlan(testPlainTrucks.Query(), nil),
			filter: &pagination.QueryOptions{Query: "frshhaul"},
		},
		{
			name:   "no filter",
			plan:   newPlan(testTrucks.Query(), nil),
			filter: nil,
		},
		{
			name:   "only short words",
			plan:   newPlan(testTrucks.Query(), nil),
			filter: &pagination.QueryOptions{Query: "dry van"},
		},
		{
			name:   "websearch syntax",
			plan:   newPlan(testTrucks.Query(), nil),
			filter: &pagination.QueryOptions{Query: `"freshhaul foods"`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, ok := tc.plan.fuzzy(tc.filter)
			assert.False(t, ok)
		})
	}
}

func TestNarrow_TheFallbackKeepsTheFiltersAndDropsTheSearch(t *testing.T) {
	t.Parallel()

	q := testTrucks.Query()
	p := newPlan(q, nil)
	req := &Request[testTruck]{Filter: &pagination.QueryOptions{
		Query: "frshhaul",
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "active", Operator: dbtype.OpEqual, Value: true},
		},
	}}

	read := narrow(p, q, req)
	require.NotNil(t, read)

	searched, _ := render(t, q.Limit(10))
	assert.Contains(t, searched, `t."search_vector" @@`)
	assert.Contains(t, searched, `t."active" = $3`)

	fallback, _ := render(t, read.unmatched.Limit(10))
	assert.NotContains(t, fallback, `t."search_vector" @@`)
	assert.Contains(t, fallback, `t."active" = $3`)

	plain := testPlainTrucks.Query()
	assert.Nil(t, narrow(newPlan(plain, nil), plain, &Request[testPlainTruck]{
		Filter: &pagination.QueryOptions{Query: "frshhaul"},
	}))
}
