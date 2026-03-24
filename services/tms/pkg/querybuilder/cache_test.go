package querybuilder

import (
	"reflect"
	"sync"
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEntity struct {
	ID     string `json:"id"     bun:"id,pk"`
	Name   string `json:"name"   bun:"name"`
	Status string `json:"status" bun:"status"`
}

func (t *testEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "te",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (t *testEntity) GetTableName() string {
	return "test_entities"
}

func TestGetFieldConfiguration_Caching(t *testing.T) {
	ClearCaches()

	entity := &testEntity{}

	config1 := GetFieldConfiguration(entity)
	config2 := GetFieldConfiguration(entity)

	assert.Same(t, config1, config2, "Should return same cached instance")
}

func TestGetFieldConfiguration_Content(t *testing.T) {
	ClearCaches()

	entity := &testEntity{}
	config := GetFieldConfiguration(entity)

	assert.True(t, config.FilterableFields["id"], "id should be filterable")
	assert.True(t, config.FilterableFields["name"], "name should be filterable")
	assert.True(t, config.FilterableFields["status"], "status should be filterable")

	assert.True(t, config.SortableFields["id"], "id should be sortable")
	assert.True(t, config.SortableFields["name"], "name should be sortable")
	assert.True(t, config.SortableFields["status"], "status should be sortable")

	assert.Equal(t, "id", config.FieldMap["id"])
	assert.Equal(t, "name", config.FieldMap["name"])
	assert.Equal(t, "status", config.FieldMap["status"])
}

func TestGetFieldConfiguration_ConcurrentAccess(t *testing.T) {
	ClearCaches()

	entity := &testEntity{}
	var wg sync.WaitGroup
	results := make([]*domaintypes.FieldConfiguration, 100)

	for i := range 100 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetFieldConfiguration(entity)
		}(i)
	}

	wg.Wait()

	first := results[0]
	for i, config := range results {
		assert.Same(
			t,
			first,
			config,
			"All goroutines should get same cached instance (index %d)",
			i,
		)
	}
}

func TestExtractFieldsFromStruct(t *testing.T) {
	ClearCaches()

	type testStruct struct {
		ID        string `json:"id"        bun:"id,pk"`
		FirstName string `json:"firstName" bun:"first_name"`
		LastName  string `json:"lastName"`
		Ignored   string `json:"-"`
		NoTag     string
	}

	fields := ExtractFieldsFromStruct(&testStruct{})

	assert.Equal(t, "id", fields["id"])
	assert.Equal(t, "first_name", fields["firstName"])
	assert.Equal(t, "last_name", fields["lastName"])
	assert.NotContains(t, fields, "Ignored")
	assert.NotContains(t, fields, "NoTag")
}

func TestExtractFieldsFromStruct_Caching(t *testing.T) {
	ClearCaches()

	type cachingTest struct {
		Field string `json:"field" bun:"field"`
	}

	fields1 := ExtractFieldsFromStruct(&cachingTest{})
	fields2 := ExtractFieldsFromStruct(&cachingTest{})

	assert.Equal(t, fields1, fields2)

	fields1["modified"] = "value"
	assert.NotContains(t, fields2, "modified", "Returned maps should be copies")
}

func TestExtractFieldsFromStruct_StaticMapperReturnsCopy(t *testing.T) {
	source := map[string]string{
		"firstName": "first_name",
		"lastName":  "last_name",
	}

	mapper := &staticMapperStub{fieldMap: source}

	result := ExtractFieldsFromStruct(mapper)
	result["injected"] = "evil"

	_, exists := source["injected"]
	assert.False(t, exists, "mutating the returned map must not corrupt the source StaticFieldMap")

	result2 := ExtractFieldsFromStruct(mapper)
	_, exists2 := result2["injected"]
	assert.False(t, exists2, "subsequent calls must not see mutations from previous calls")
}

type staticMapperStub struct {
	fieldMap map[string]string
}

func (s *staticMapperStub) GetStaticFieldMap() map[string]string {
	return s.fieldMap
}

func TestExtractFieldsFromStruct_ConcurrentAccess(t *testing.T) {
	ClearCaches()

	type concurrentStruct struct {
		ID   string `json:"id"   bun:"id"`
		Name string `json:"name" bun:"name"`
	}

	var wg sync.WaitGroup
	results := make([]map[string]string, 100)

	for i := range 100 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = ExtractFieldsFromStruct(&concurrentStruct{})
		}(i)
	}

	wg.Wait()

	for i, fields := range results {
		assert.Equal(t, "id", fields["id"], "Index %d: id field mismatch", i)
		assert.Equal(t, "name", fields["name"], "Index %d: name field mismatch", i)
	}
}

func TestExtractFieldsFromStruct_NonStruct(t *testing.T) {
	fields := ExtractFieldsFromStruct("not a struct")
	assert.Empty(t, fields)

	fields = ExtractFieldsFromStruct(123)
	assert.Empty(t, fields)
}

func TestClearCaches(t *testing.T) {
	entity := &testEntity{}
	_ = GetFieldConfiguration(entity)
	_ = ExtractFieldsFromStruct(entity)

	fieldMapCacheMu.RLock()
	hasFieldMap := len(fieldMapCache) > 0
	fieldMapCacheMu.RUnlock()

	fieldConfigCacheMu.RLock()
	hasFieldConfig := len(fieldConfigCache) > 0
	fieldConfigCacheMu.RUnlock()

	require.True(t, hasFieldMap || hasFieldConfig, "Caches should have data before clearing")

	ClearCaches()

	fieldMapCacheMu.RLock()
	fieldMapEmpty := len(fieldMapCache) == 0
	fieldMapCacheMu.RUnlock()

	fieldConfigCacheMu.RLock()
	fieldConfigEmpty := len(fieldConfigCache) == 0
	fieldConfigCacheMu.RUnlock()

	assert.True(t, fieldMapEmpty, "Field map cache should be empty after clearing")
	assert.True(t, fieldConfigEmpty, "Field config cache should be empty after clearing")
}

func TestWarmFieldConfigCache(t *testing.T) {
	ClearCaches()

	entity := &testEntity{}

	fieldConfigCacheMu.RLock()
	_, exists := fieldConfigCache[reflect.TypeFor[testEntity]()]
	fieldConfigCacheMu.RUnlock()
	require.False(t, exists, "Cache should be empty before warming")

	WarmFieldConfigCache(entity)

	fieldConfigCacheMu.RLock()
	_, exists = fieldConfigCache[reflect.TypeFor[testEntity]()]
	fieldConfigCacheMu.RUnlock()
	assert.True(t, exists, "Cache should contain entity after warming")
}
