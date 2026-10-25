package keelcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTablesAreFoundByNameAndModel(t *testing.T) {
	t.Parallel()

	customers := MustNamed("customers")
	assert.Equal(t, "customers", customers.Name)

	byModel, ok := ForModel(customers.Model)
	require.True(t, ok)
	assert.Same(t, customers, byModel)

	_, ok = ForModel("nothing.Nothing")
	assert.False(t, ok)
	assert.Panics(t, func() { MustNamed("no_such_table") })
}
