package customfield

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsResourceTypeSupported_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resourceType string
	}{
		{name: "trailer", resourceType: "trailer"},
		{name: "worker", resourceType: "worker"},
		{name: "shipment", resourceType: "shipment"},
		{name: "customer", resourceType: "customer"},
		{name: "location", resourceType: "location"},
		{name: "tractor", resourceType: "tractor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsResourceTypeSupported(tt.resourceType))
		})
	}
}

func TestIsResourceTypeSupported_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resourceType string
	}{
		{name: "empty string", resourceType: ""},
		{name: "unknown type", resourceType: "unknown"},
		{name: "uppercase", resourceType: "TRAILER"},
		{name: "mixed case", resourceType: "Trailer"},
		{name: "with spaces", resourceType: "trailer "},
		{name: "typo", resourceType: "trailerr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsResourceTypeSupported(tt.resourceType))
		})
	}
}

func TestGetSupportedResourceTypes(t *testing.T) {
	t.Parallel()

	result := GetSupportedResourceTypes()

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "trailer")
	assert.Contains(t, result, "worker")
	assert.Contains(t, result, "shipment")
	assert.Contains(t, result, "customer")
	assert.Contains(t, result, "location")
	assert.Contains(t, result, "tractor")

	for i := 1; i < len(result); i++ {
		assert.True(t, result[i-1] < result[i], "result should be sorted alphabetically")
	}
}

func TestRegisterResourceType(t *testing.T) {
	RegisterResourceType("test_resource")
	defer func() {
		delete(supportedResourceTypes, "test_resource")
	}()

	assert.True(t, IsResourceTypeSupported("test_resource"))
	assert.Contains(t, GetSupportedResourceTypes(), "test_resource")
}

func TestRegisterResourceType_Idempotent(t *testing.T) {
	RegisterResourceType("idempotent_test")
	RegisterResourceType("idempotent_test")
	defer func() {
		delete(supportedResourceTypes, "idempotent_test")
	}()

	assert.True(t, IsResourceTypeSupported("idempotent_test"))
}
