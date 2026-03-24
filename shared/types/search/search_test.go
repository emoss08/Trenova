package search_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityTypeConstants(t *testing.T) {
	t.Parallel()

	t.Run("shipment entity type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, search.EntityType("shipment"), search.EntityTypeShipment)
	})

	t.Run("customer entity type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, search.EntityType("customer"), search.EntityTypeCustomer)
	})
}

func TestDocument_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid document", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{
			ID:             "doc_123",
			EntityType:     search.EntityTypeShipment,
			OrganizationID: "org_456",
			BusinessUnitID: "bu_789",
			Title:          "Test Document",
			Subtitle:       "Optional subtitle",
			Content:        "Some content",
			Metadata:       map[string]any{"key": "value"},
			CreatedAt:      1234567890,
			UpdatedAt:      1234567890,
		}

		err := doc.Validate()
		require.NoError(t, err)
	})

	t.Run("missing ID", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{
			EntityType:     search.EntityTypeShipment,
			OrganizationID: "org_456",
			BusinessUnitID: "bu_789",
			Title:          "Test Document",
		}

		err := doc.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ID is required")
	})

	t.Run("missing entity type", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{
			ID:             "doc_123",
			OrganizationID: "org_456",
			BusinessUnitID: "bu_789",
			Title:          "Test Document",
		}

		err := doc.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Entity type is required")
	})

	t.Run("missing organization ID", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{
			ID:             "doc_123",
			EntityType:     search.EntityTypeShipment,
			BusinessUnitID: "bu_789",
			Title:          "Test Document",
		}

		err := doc.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Organization ID is required")
	})

	t.Run("missing business unit ID", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{
			ID:             "doc_123",
			EntityType:     search.EntityTypeShipment,
			OrganizationID: "org_456",
			Title:          "Test Document",
		}

		err := doc.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Business unit ID is required")
	})

	t.Run("missing title", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{
			ID:             "doc_123",
			EntityType:     search.EntityTypeShipment,
			OrganizationID: "org_456",
			BusinessUnitID: "bu_789",
		}

		err := doc.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Title is required")
	})

	t.Run("optional fields can be empty", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{
			ID:             "doc_123",
			EntityType:     search.EntityTypeCustomer,
			OrganizationID: "org_456",
			BusinessUnitID: "bu_789",
			Title:          "Minimal Document",
		}

		err := doc.Validate()
		require.NoError(t, err)
	})

	t.Run("multiple missing fields", func(t *testing.T) {
		t.Parallel()
		doc := &search.Document{}

		err := doc.Validate()
		require.Error(t, err)
	})
}
