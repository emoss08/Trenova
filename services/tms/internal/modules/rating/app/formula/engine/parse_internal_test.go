package engine

import (
	"testing"

	"github.com/expr-lang/expr/parser"
	"github.com/stretchr/testify/assert"
)

func TestReferencedPaths_ParsesWithTheEnginesBuiltins(t *testing.T) {
	t.Parallel()

	expression := `sum(customer.code == "ACME" ? 10 : 0, stopFee, 25)`

	_, bareErr := parser.Parse(expression)
	paths := referencedPaths(expression)

	assert.Error(t, bareErr, "the bare parser is what this guards against")
	assert.Contains(t, paths, "customer.code")
	assert.Contains(t, paths, "stopFee")
}

func TestExtractLookupTables_ParsesTheEnginesOwnFunctions(t *testing.T) {
	t.Parallel()

	tables, err := ExtractLookupTables(`sum(lookup("fsc", fuelPrice) * linehaul, stopFee, 25)`)
	assert.NoError(t, err)
	assert.Equal(t, []string{"fsc"}, tables)
}
