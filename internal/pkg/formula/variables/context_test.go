package variables_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultVariableContext_GetEntity(t *testing.T) {
	s := &shipment.Shipment{
		ProNumber: "TEST123",
	}

	ctx := variables.NewDefaultContext(s, nil)
	entity := ctx.GetEntity()

	shipmentEntity, ok := entity.(*shipment.Shipment)
	require.True(t, ok)
	assert.Equal(t, "TEST123", shipmentEntity.ProNumber)
}

func TestDefaultVariableContext_GetField(t *testing.T) {
	// * Create test shipment
	s := &shipment.Shipment{
		ProNumber: "TEST123",
		Customer: &customer.Customer{
			Name: "Test Customer",
		},
	}

	// * Create resolver
	resolver := schema.NewDefaultDataResolver()

	// * Create context
	ctx := variables.NewDefaultContext(s, resolver)

	// * Get simple field
	val, err := ctx.GetField("ProNumber")
	require.NoError(t, err)
	assert.Equal(t, "TEST123", val)

	// * Get nested field
	val, err = ctx.GetField("Customer.Name")
	require.NoError(t, err)
	assert.Equal(t, "Test Customer", val)

	// * No resolver should error
	ctxNoResolver := variables.NewDefaultContext(s, nil)
	_, err = ctxNoResolver.GetField("ProNumber")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no resolver configured")
}

func TestDefaultVariableContext_GetComputed(t *testing.T) {
	s := &shipment.Shipment{}

	// * Create resolver with compute function
	resolver := schema.NewDefaultDataResolver()
	resolver.RegisterComputer("testCompute", func(entity any) (any, error) {
		return "computed value", nil
	})

	// * Create context
	ctx := variables.NewDefaultContext(s, resolver)

	// * Get computed value
	val, err := ctx.GetComputed("testCompute")
	require.NoError(t, err)
	assert.Equal(t, "computed value", val)

	// * Non-existent compute function
	_, err = ctx.GetComputed("nonExistent")
	assert.Error(t, err)
}

func TestDefaultVariableContext_Metadata(t *testing.T) {
	s := &shipment.Shipment{}

	// * Create context without metadata
	ctx1 := variables.NewDefaultContext(s, nil)
	meta1 := ctx1.GetMetadata()
	assert.NotNil(t, meta1)
	assert.Empty(t, meta1)

	// * Set metadata
	ctx1.SetMetadata("key1", "value1")
	assert.Equal(t, "value1", ctx1.GetMetadata()["key1"])

	// * Create context with metadata
	initialMeta := map[string]any{
		"key2": "value2",
		"key3": 123,
	}
	ctx2 := variables.NewDefaultContextWithMetadata(s, nil, initialMeta)
	meta2 := ctx2.GetMetadata()
	assert.Equal(t, "value2", meta2["key2"])
	assert.Equal(t, 123, meta2["key3"])
}

func TestDefaultVariableContext_WithMethods(t *testing.T) {
	s1 := &shipment.Shipment{ProNumber: "TEST1"}
	s2 := &shipment.Shipment{ProNumber: "TEST2"}
	resolver1 := schema.NewDefaultDataResolver()
	resolver2 := schema.NewDefaultDataResolver()

	ctx := variables.NewDefaultContext(s1, resolver1)

	// * WithEntity
	newCtx := ctx.WithEntity(s2)
	assert.Equal(t, s2, newCtx.GetEntity())
	assert.Equal(t, s1, ctx.GetEntity()) // Original unchanged

	// * WithResolver
	newCtx2 := ctx.WithResolver(resolver2)
	val, _ := newCtx2.GetField("ProNumber")
	assert.Equal(t, "TEST1", val) // Still using original entity
}
