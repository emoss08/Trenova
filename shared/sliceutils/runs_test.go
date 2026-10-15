package sliceutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/stretchr/testify/assert"
)

type scoredName struct {
	name  string
	score int
}

func scoreOf(item scoredName) int   { return item.score }
func nameOf(item scoredName) string { return item.name }

func TestRunsGroupsNeighboursThatShareAKey(t *testing.T) {
	t.Parallel()

	runs := sliceutils.Runs([]scoredName{
		{"a", 4}, {"b", 4}, {"c", 3}, {"d", 4},
	}, scoreOf, nameOf)

	assert.Equal(t, [][]string{{"a", "b"}, {"c"}, {"d"}}, runs)
}

func TestRunsOfOneItemIsOneRun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, [][]string{{"a"}}, sliceutils.Runs([]scoredName{{"a", 1}}, scoreOf, nameOf))
}

func TestRunsOfNothingIsNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, sliceutils.Runs[scoredName](nil, scoreOf, nameOf))
}
