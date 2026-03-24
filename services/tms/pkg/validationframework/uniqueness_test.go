package validationframework

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniquenessRequest(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	excludeID := pulid.MustNew("ent_")

	req := &UniquenessRequest{
		TableName:      "test_table",
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ExcludeID:      excludeID,
		ScopeFields: []FieldCheck{
			{Column: "business_unit_id", Value: buID, CaseSensitive: true},
		},
		Fields: []FieldCheck{
			{Column: "name", Value: "Test", CaseSensitive: false},
		},
	}

	assert.Equal(t, "test_table", req.TableName)
	assert.Equal(t, orgID, req.OrganizationID)
	assert.Equal(t, buID, req.BusinessUnitID)
	assert.Equal(t, excludeID, req.ExcludeID)
	require.Len(t, req.ScopeFields, 1)
	assert.Equal(t, "business_unit_id", req.ScopeFields[0].Column)
	require.Len(t, req.Fields, 1)
	assert.Equal(t, "name", req.Fields[0].Column)
	assert.Equal(t, "Test", req.Fields[0].Value)
	assert.False(t, req.Fields[0].CaseSensitive)
}

func TestFieldCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		field         FieldCheck
		expectedCol   string
		expectedValue any
		expectedCase  bool
	}{
		{
			name:          "case insensitive field",
			field:         FieldCheck{Column: "name", Value: "Test", CaseSensitive: false},
			expectedCol:   "name",
			expectedValue: "Test",
			expectedCase:  false,
		},
		{
			name:          "case sensitive field",
			field:         FieldCheck{Column: "code", Value: "ABC123", CaseSensitive: true},
			expectedCol:   "code",
			expectedValue: "ABC123",
			expectedCase:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedCol, tt.field.Column)
			assert.Equal(t, tt.expectedValue, tt.field.Value)
			assert.Equal(t, tt.expectedCase, tt.field.CaseSensitive)
		})
	}
}

func TestNewBunUniquenessChecker(t *testing.T) {
	t.Parallel()

	checker := NewBunUniquenessChecker(nil)
	require.NotNil(t, checker)
}

func TestBunUniquenessChecker_CheckUniqueness_ValidationErrors(t *testing.T) {
	t.Parallel()

	checker := NewBunUniquenessChecker(nil)

	t.Run("empty table name", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName: "",
			Fields:    []FieldCheck{{Column: "name", Value: "test"}},
		}

		_, err := checker.CheckUniqueness(t.Context(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "table name is required")
	})

	t.Run("empty fields", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName: "test_table",
			Fields:    []FieldCheck{},
		}

		_, err := checker.CheckUniqueness(t.Context(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one field is required")
	})
}

func TestMockUniquenessChecker(t *testing.T) {
	t.Parallel()

	t.Run("returns false when no conflict", func(t *testing.T) {
		checker := newMockUniquenessChecker()
		req := &UniquenessRequest{
			TableName: "test_table",
			Fields:    []FieldCheck{{Column: "name", Value: "test"}},
		}

		checker.On("CheckUniqueness", t.Context(), req).Return(false, nil)

		exists, err := checker.CheckUniqueness(t.Context(), req)

		require.NoError(t, err)
		assert.False(t, exists)
		checker.AssertExpectations(t)
	})

	t.Run("returns true when conflict exists", func(t *testing.T) {
		checker := newMockUniquenessChecker()
		req := &UniquenessRequest{
			TableName: "test_table",
			Fields:    []FieldCheck{{Column: "name", Value: "test"}},
		}

		checker.On("CheckUniqueness", t.Context(), req).Return(true, nil)

		exists, err := checker.CheckUniqueness(t.Context(), req)

		require.NoError(t, err)
		assert.True(t, exists)
		checker.AssertExpectations(t)
	})
}
