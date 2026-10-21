package searchfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuote_EscapesQuotesAndBackslashes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `"plain"`, Quote("plain"))
	assert.Equal(t, `"a\"b"`, Quote(`a"b`))
	assert.Equal(t, `"a\\\" OR x = \"y"`, Quote(`a\" OR x = "y`))
}

func TestAnd_KeepsAnInjectedOrInsideItsOwnCondition(t *testing.T) {
	t.Parallel()

	filter := And(
		Eq("organization_id", "org_1"),
		Eq("resource_id", `x" OR organization_id = "org_2`),
	)
	assert.Equal(
		t,
		`(organization_id = "org_1") AND (resource_id = "x\" OR organization_id = \"org_2")`,
		filter,
	)
}

func TestEq_RefusesAFieldThatIsNotAPlainName(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { Eq("id = 1 OR id", "x") })
}
