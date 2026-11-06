package searchtext_test

import (
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
