package listquery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescribe_TheFieldsAListAnswersTo(t *testing.T) {
	t.Parallel()

	got := Describe(new(testTruck), testTrucks.Ref.Table)

	assert.True(t, got.FilterableFields["code"])
	assert.True(t, got.FilterableFields["year"])
	assert.True(t, got.FilterableFields["owner.name"])
	assert.False(t, got.FilterableFields["searchVector"])
	assert.False(t, got.FilterableFields["token"])
	assert.False(t, got.FilterableFields["loads.name"])

	assert.True(t, got.SortableFields["code"])
	assert.False(t, got.SortableFields["owner.name"], "a keyset page orders by its own columns")

	assert.True(t, got.EnumMap["status"])
	assert.False(t, got.EnumMap["code"])

	assert.Equal(t, "created_at", got.FieldMap["createdAt"])
	assert.NotContains(t, got.FieldMap, "owner.name")

	assert.True(t, got.NonNullableFields["code"])
	assert.False(t, got.NonNullableFields["year"])
	assert.True(t, got.IntegerFields["year"])
	assert.False(t, got.IntegerFields["weight"])
}

func TestDescribe_WithoutAModelOrTable(t *testing.T) {
	t.Parallel()

	assert.Nil(t, Describe(nil, testTrucks.Ref.Table))
	assert.Nil(t, Describe(new(testTruck), nil))
}
