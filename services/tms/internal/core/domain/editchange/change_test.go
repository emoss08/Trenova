package editchange

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type setting struct {
	name  string
	limit int
}

var rules = []Rule[setting]{
	{Field: "name", Label: "Name", Same: func(a, b *setting) bool { return a.name == b.name }},
	{Field: "limit", Label: "Limit", Same: func(a, b *setting) bool { return a.limit == b.limit }},
}

func TestDetectListsDifferingSettingsInRuleOrder(t *testing.T) {
	t.Parallel()

	changes := Detect(rules, &setting{name: "a", limit: 1}, &setting{name: "b", limit: 2})

	assert.Equal(t, []Change{{Field: "name", Label: "Name"}, {Field: "limit", Label: "Limit"}}, changes)
	assert.Equal(t, "Name, Limit", Summary(changes))
}

func TestDetectHasNothingToCompareWithoutBothSides(t *testing.T) {
	t.Parallel()

	assert.Nil(t, Detect(rules, nil, &setting{}))
	assert.Empty(t, Detect(rules, &setting{name: "a"}, &setting{name: "a"}))
	assert.Empty(t, Summary(nil))
}
