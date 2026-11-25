package searchtext_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/searchtext"
	"github.com/stretchr/testify/assert"
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

			got := searchtext.FuzzyWords(tc.query)
			if len(tc.want) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHasOperators_RecognizesWebsearchSyntax(t *testing.T) {
	t.Parallel()

	assert.True(t, searchtext.HasOperators(`"dry van"`))
	assert.True(t, searchtext.HasOperators("dry -van"))
	assert.True(t, searchtext.HasOperators("dry OR van"))
	assert.False(t, searchtext.HasOperators("dry van"))
	assert.False(t, searchtext.HasOperators("store - main"), "a lone dash is not an exclusion")
}

func TestIdentifierTerms_SplitsAListOfIdentifiers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		query string
		want  []string
	}{
		{"S261 S262", []string{"S261", "S262"}},
		{"PRO-1001, PRO-1002;BOL/77 | 4410", []string{"PRO-1001", "PRO-1002", "BOL/77", "4410"}},
		{"S261 S261", nil},
		{"S261", nil},
		{"dry van", nil},
		{"store 1123", nil},
		{"S261 or S262", nil},
		{`"S261 S262"`, nil},
		{"S261 a1", nil},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, searchtext.IdentifierTerms(tc.query))
		})
	}
}

func TestIdentifierTerms_IsBounded(t *testing.T) {
	t.Parallel()

	terms := make([]string, 0, searchtext.MaxIdentifierTerms+1)
	for i := range searchtext.MaxIdentifierTerms + 1 {
		terms = append(terms, "S"+strconv.Itoa(1000+i))
	}

	assert.Nil(t, searchtext.IdentifierTerms(strings.Join(terms, " ")))
	assert.Len(t, searchtext.IdentifierTerms(strings.Join(terms[1:], " ")),
		searchtext.MaxIdentifierTerms)
}

func TestVectorQuery_MatchesAnyListedIdentifier(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "S261 or S262", searchtext.VectorQuery("S261  S262"))
	assert.Equal(t, "dry van", searchtext.VectorQuery("dry van"))
}
