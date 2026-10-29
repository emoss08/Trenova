package reporting

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalog_DecidesEveryEntityForTheCaller(t *testing.T) {
	t.Parallel()

	svc, engine := engineService(t, engineOptions{denied: []permission.Resource{
		permission.ResourceCustomer,
	}})

	request := planRequest(testUserID())
	catalog, err := svc.Catalog(t.Context(), &request)
	require.NoError(t, err)

	assert.Equal(t, engine.Catalog().Version(), catalog.Version)
	assert.NotEmpty(t, catalog.Rules.Aggregations)
	require.Len(t, catalog.Entities, len(engine.Catalog().Entities()))
	byKey := make(map[string]*CatalogEntity, len(catalog.Entities))
	for i := range catalog.Entities {
		byKey[catalog.Entities[i].View.Key] = &catalog.Entities[i]
	}

	shipment := byKey["shipment"]
	require.NotNil(t, shipment)
	assert.True(t, shipment.Readable)
	assert.True(t, shipment.Exportable)
	customer := byKey["customer"]
	require.NotNil(t, customer)
	assert.False(t, customer.Readable)
	assert.True(t, customer.Usable(), "a customer's reference fields stay reachable through a join")
	require.Len(t, customer.Fields, len(customer.View.Fields))
	require.Len(t, customer.Edges, len(customer.View.Edges))
}
