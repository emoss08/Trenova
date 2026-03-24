package querybuilder

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type mockEntity struct {
	ID     string `json:"id"     bun:"id,pk"`
	Name   string `json:"name"   bun:"name"`
	Status string `json:"status" bun:"status"`
	Code   string `json:"code"   bun:"code"`
}

func (m *mockEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "m",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "code", Type: domaintypes.FieldTypeText},
		},
	}
}

func (m *mockEntity) GetTableName() string {
	return "mock_entities"
}

func newTestDB() *bun.DB {
	return bun.NewDB(&sql.DB{}, pgdialect.New())
}

func newTestQuery(db *bun.DB) *bun.SelectQuery {
	return db.NewSelect().Model((*mockEntity)(nil)).ModelTableExpr("mock_entities AS m")
}

func querySQL(t *testing.T, query *bun.SelectQuery) string {
	t.Helper()
	return query.String()
}

func TestQueryBuilder_MaxFilters(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := make([]domaintypes.FieldFilter, MaxFilters+10)
	for i := range filters {
		filters[i] = domaintypes.FieldFilter{
			Field:    "name",
			Operator: dbtype.OpEqual,
			Value:    "test",
		}
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.Equal(t, 50, MaxFilters, "MaxFilters should be 50")
}

func TestQueryBuilder_MaxSortFields(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	sorts := make([]domaintypes.SortField, MaxSortFields+5)
	for i := range sorts {
		sorts[i] = domaintypes.SortField{
			Field:     "name",
			Direction: dbtype.SortDirectionAsc,
		}
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplySort(sorts)

	assert.Equal(t, 10, MaxSortFields, "MaxSortFields should be 10")
}

func TestQueryBuilder_InvalidFields_Filters(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
		{Field: "invalid_field", Operator: dbtype.OpEqual, Value: "test"},
		{Field: "another_invalid", Operator: dbtype.OpEqual, Value: "test"},
		{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.True(t, qb.HasInvalidFields())
	invalidFields := qb.GetInvalidFields()
	assert.Len(t, invalidFields, 2)
	assert.Contains(t, invalidFields, "invalid_field")
	assert.Contains(t, invalidFields, "another_invalid")
}

func TestQueryBuilder_InvalidFields_Sorts(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	sorts := []domaintypes.SortField{
		{Field: "name", Direction: dbtype.SortDirectionAsc},
		{Field: "nonexistent", Direction: dbtype.SortDirectionDesc},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplySort(sorts)

	assert.True(t, qb.HasInvalidFields())
	invalidFields := qb.GetInvalidFields()
	assert.Contains(t, invalidFields, "nonexistent")
}

func TestQueryBuilder_InvalidSortDirection_IsRejected(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplySort([]domaintypes.SortField{
		{Field: "name", Direction: dbtype.SortDirection("desc; drop table users; --")},
	})

	assert.True(t, qb.HasInvalidFields())
	assert.Contains(t, qb.GetInvalidFields(), "name")
	assert.NotContains(t, strings.ToLower(querySQL(t, qb.GetQuery())), "drop table")
}

func TestQueryBuilder_NoInvalidFields(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
		{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.False(t, qb.HasInvalidFields())
	assert.Empty(t, qb.GetInvalidFields())
}

func TestQueryBuilder_FilterOperators(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	tests := []struct {
		name     string
		operator dbtype.Operator
		value    any
	}{
		{"Equal", dbtype.OpEqual, "test"},
		{"NotEqual", dbtype.OpNotEqual, "test"},
		{"Contains", dbtype.OpContains, "test"},
		{"StartsWith", dbtype.OpStartsWith, "test"},
		{"EndsWith", dbtype.OpEndsWith, "test"},
		{"In", dbtype.OpIn, []string{"a", "b"}},
		{"NotIn", dbtype.OpNotIn, []string{"a", "b"}},
		{"IsNull", dbtype.OpIsNull, nil},
		{"IsNotNull", dbtype.OpIsNotNull, nil},
		{"GreaterThan", dbtype.OpGreaterThan, 10},
		{"LessThan", dbtype.OpLessThan, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := newTestQuery(db)
			qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

			filters := []domaintypes.FieldFilter{
				{Field: "name", Operator: tt.operator, Value: tt.value},
			}

			qb.ApplyFilters(filters)
			assert.NotNil(t, qb.GetQuery())
		})
	}
}

func TestQueryBuilder_SortDirections(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	tests := []struct {
		name      string
		direction dbtype.SortDirection
	}{
		{"Ascending", dbtype.SortDirectionAsc},
		{"Descending", dbtype.SortDirectionDesc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := newTestQuery(db)
			qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

			sorts := []domaintypes.SortField{
				{Field: "name", Direction: tt.direction},
			}

			qb.ApplySort(sorts)
			assert.NotNil(t, qb.GetQuery())
		})
	}
}

func TestQueryBuilder_Chaining(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	result := qb.
		WithTraversalSupport(true).
		ApplyFilters([]domaintypes.FieldFilter{
			{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
		}).
		ApplySort([]domaintypes.SortField{
			{Field: "name", Direction: dbtype.SortDirectionAsc},
		}).
		GetQuery()

	assert.NotNil(t, result)
}

func TestQueryBuilder_EmptyFiltersAndSorts(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	qb.ApplyFilters(nil)
	qb.ApplySort(nil)

	assert.False(t, qb.HasInvalidFields())
	assert.NotNil(t, qb.GetQuery())
}

func TestNewQueryBuilder(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	query := newTestQuery(db)
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	qb := New(query, "m", fieldConfig)

	assert.NotNil(t, qb)
	assert.Equal(t, "m", qb.tableAlias)
	assert.NotNil(t, qb.fieldConfig)
	assert.False(t, qb.traversalEnabled)
	assert.Empty(t, qb.invalidFields)
}

func TestNewWithPostgresSearch(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)
	query := newTestQuery(db)

	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	assert.NotNil(t, qb)
	assert.Equal(t, "m", qb.tableAlias)
	assert.NotNil(t, qb.fieldConfig)
	assert.NotNil(t, qb.searchConfig)
	assert.Empty(t, qb.invalidFields)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 50, MaxFilters)
	assert.Equal(t, 10, MaxSortFields)
}

func TestQueryBuilder_FilterTruncation(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := make([]domaintypes.FieldFilter, 100)
	for i := range filters {
		filters[i] = domaintypes.FieldFilter{
			Field:    "name",
			Operator: dbtype.OpEqual,
			Value:    "test",
		}
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_SortTruncation(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	sorts := make([]domaintypes.SortField, 20)
	for i := range sorts {
		sorts[i] = domaintypes.SortField{
			Field:     "name",
			Direction: dbtype.SortDirectionAsc,
		}
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplySort(sorts)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_MixedValidAndInvalidFields(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpEqual, Value: "test1"},
		{Field: "bad_field", Operator: dbtype.OpEqual, Value: "bad"},
		{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
		{Field: "another_bad", Operator: dbtype.OpEqual, Value: "bad"},
		{Field: "code", Operator: dbtype.OpEqual, Value: "ABC"},
	}

	sorts := []domaintypes.SortField{
		{Field: "name", Direction: dbtype.SortDirectionAsc},
		{Field: "invalid_sort", Direction: dbtype.SortDirectionDesc},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)
	qb.ApplySort(sorts)

	assert.True(t, qb.HasInvalidFields())
	invalidFields := qb.GetInvalidFields()
	assert.Len(t, invalidFields, 3)
	assert.Contains(t, invalidFields, "bad_field")
	assert.Contains(t, invalidFields, "another_bad")
	assert.Contains(t, invalidFields, "invalid_sort")
}

func TestQueryBuilder_WithTraversalSupport(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	assert.False(t, qb.traversalEnabled)

	qb.WithTraversalSupport(true)
	assert.True(t, qb.traversalEnabled)

	qb.WithTraversalSupport(false)
	assert.False(t, qb.traversalEnabled)
}

func TestQueryBuilder_DateRangeFilter(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{
			Field:    "name",
			Operator: dbtype.OpDateRange,
			Value: map[string]any{
				"start": "2024-01-01",
				"end":   "2024-12-31",
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_DateRangeFilter_StartOnly(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{
			Field:    "name",
			Operator: dbtype.OpDateRange,
			Value: map[string]any{
				"start": "2024-01-01",
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_DateRangeFilter_EndOnly(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{
			Field:    "name",
			Operator: dbtype.OpDateRange,
			Value: map[string]any{
				"end": "2024-12-31",
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_DateRangeFilter_InvalidValue(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{
			Field:    "name",
			Operator: dbtype.OpDateRange,
			Value:    "not a map",
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_DateRangeFilter_InvalidDateFormat(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{
			Field:    "name",
			Operator: dbtype.OpDateRange,
			Value: map[string]any{
				"start": "invalid-date",
				"end":   "also-invalid",
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_EnumFieldFilter(t *testing.T) {
	ClearCaches()

	db := newTestDB()

	type enumEntity struct {
		ID     string `json:"id"     bun:"id,pk"`
		Status string `json:"status" bun:"status"`
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"status": true},
		SortableFields:   map[string]bool{"status": true},
		FieldMap:         map[string]string{"status": "status"},
		EnumMap:          map[string]bool{"status": true},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	filters := []domaintypes.FieldFilter{
		{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
		{Field: "status", Operator: dbtype.OpContains, Value: "act"},
		{Field: "status", Operator: dbtype.OpIn, Value: []string{"active", "pending"}},
	}

	query := db.NewSelect().Model((*enumEntity)(nil)).ModelTableExpr("enum_entities AS ee")

	searchConfig := domaintypes.PostgresSearchConfig{
		TableAlias: "ee",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}

	qb := &QueryBuilder{
		query:         query,
		tableAlias:    "ee",
		fieldConfig:   fieldConfig,
		searchConfig:  &searchConfig,
		appliedJoins:  make(map[string]bool),
		invalidFields: make([]string, 0),
	}

	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_EmptyEnumValue(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)
	fieldConfig.EnumMap["status"] = true

	filters := []domaintypes.FieldFilter{
		{Field: "status", Operator: dbtype.OpEqual, Value: ""},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

type nestedTestEntity struct {
	ID         string `json:"id"         bun:"id,pk"`
	Name       string `json:"name"       bun:"name"`
	CustomerID string `json:"customerId" bun:"customer_id"`
}

type customerEntity struct {
	ID   string `json:"id"   bun:"id,pk"`
	Name string `json:"name" bun:"name"`
	Code string `json:"code" bun:"code"`
}

func (n *nestedTestEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "nte",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "customer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: &customerEntity{},
				TargetTable:  "customers",
				ForeignKey:   "customer_id",
				ReferenceKey: "id",
				Alias:        "cust",
				Queryable:    true,
			},
		},
	}
}

func (n *nestedTestEntity) GetTableName() string {
	return "nested_test_entities"
}

func TestQueryBuilder_TraversalFilter(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &nestedTestEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "customer.name", Operator: dbtype.OpEqual, Value: "Acme Corp"},
	}

	query := db.NewSelect().
		Model((*nestedTestEntity)(nil)).
		ModelTableExpr("nested_test_entities AS nte")
	qb := NewWithPostgresSearch(query, "nte", fieldConfig, entity)
	qb.WithTraversalSupport(true)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TraversalSort(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &nestedTestEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	sorts := []domaintypes.SortField{
		{Field: "customer.name", Direction: dbtype.SortDirectionAsc},
	}

	query := db.NewSelect().
		Model((*nestedTestEntity)(nil)).
		ModelTableExpr("nested_test_entities AS nte")
	qb := NewWithPostgresSearch(query, "nte", fieldConfig, entity)
	qb.WithTraversalSupport(true)
	qb.ApplySort(sorts)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_NestedFieldDefinition(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{
			"name":          true,
			"customer.name": true,
		},
		SortableFields: map[string]bool{
			"name":          true,
			"customer.name": true,
		},
		FieldMap: map[string]string{
			"name": "name",
		},
		EnumMap: map[string]bool{},
		NestedFields: map[string]domaintypes.NestedFieldDefintion{
			"customer.name": {
				DatabaseField: "cust.name",
				RequiredJoins: []domaintypes.JoinStep{
					{
						Table:     "customers",
						Alias:     "cust",
						Condition: "m.customer_id = cust.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
				IsEnum: false,
			},
		},
	}

	filters := []domaintypes.FieldFilter{
		{Field: "customer.name", Operator: dbtype.OpContains, Value: "Acme"},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_NestedFieldSort(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"customer.name": true},
		SortableFields:   map[string]bool{"customer.name": true},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields: map[string]domaintypes.NestedFieldDefintion{
			"customer.name": {
				DatabaseField: "cust.name",
				RequiredJoins: []domaintypes.JoinStep{
					{
						Table:     "customers",
						Alias:     "cust",
						Condition: "m.customer_id = cust.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
		},
	}

	sorts := []domaintypes.SortField{
		{Field: "customer.name", Direction: dbtype.SortDirectionDesc},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplySort(sorts)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_JoinDeduplication(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{
			"customer.name": true,
			"customer.code": true,
		},
		SortableFields: map[string]bool{
			"customer.name": true,
		},
		FieldMap: map[string]string{},
		EnumMap:  map[string]bool{},
		NestedFields: map[string]domaintypes.NestedFieldDefintion{
			"customer.name": {
				DatabaseField: "cust.name",
				RequiredJoins: []domaintypes.JoinStep{
					{
						Table:     "customers",
						Alias:     "cust",
						Condition: "m.customer_id = cust.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
			"customer.code": {
				DatabaseField: "cust.code",
				RequiredJoins: []domaintypes.JoinStep{
					{
						Table:     "customers",
						Alias:     "cust",
						Condition: "m.customer_id = cust.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
		},
	}

	filters := []domaintypes.FieldFilter{
		{Field: "customer.name", Operator: dbtype.OpEqual, Value: "Acme"},
		{Field: "customer.code", Operator: dbtype.OpEqual, Value: "ACM"},
	}

	sorts := []domaintypes.SortField{
		{Field: "customer.name", Direction: dbtype.SortDirectionAsc},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)
	qb.ApplySort(sorts)

	assert.NotNil(t, qb.GetQuery())
}

type searchVectorEntity struct {
	ID           string `json:"id"           bun:"id,pk"`
	Name         string `json:"name"         bun:"name"`
	SearchVector string `json:"searchVector" bun:"search_vector"`
}

func (s *searchVectorEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:         "sve",
		UseSearchVector:    true,
		SearchVectorColumn: "search_vector",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (s *searchVectorEntity) GetTableName() string {
	return "search_vector_entities"
}

func TestQueryBuilder_TextSearch_Simple(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyTextSearch("search term", []string{"name", "code"})

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TextSearch_EmptyQuery(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyTextSearch("", []string{"name"})

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TextSearch_NoFields(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyTextSearch("search", []string{})

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TextSearch_WithSearchVector(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &searchVectorEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := db.NewSelect().
		Model((*searchVectorEntity)(nil)).
		ModelTableExpr("search_vector_entities AS sve")
	qb := NewWithPostgresSearch(query, "sve", fieldConfig, entity)
	qb.ApplyTextSearch("test query", []string{"name"})

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TextSearch_AdvancedSyntax(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &searchVectorEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	tests := []struct {
		name  string
		query string
	}{
		{"QuotedPhrase", `"exact phrase"`},
		{"OrOperator", "word1 OR word2"},
		{"Negation", "-excluded"},
		{"Combined", `"phrase" OR other -excluded`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := db.NewSelect().
				Model((*searchVectorEntity)(nil)).
				ModelTableExpr("search_vector_entities AS sve")
			qb := NewWithPostgresSearch(query, "sve", fieldConfig, entity)
			qb.ApplyTextSearch(tt.query, []string{"name"})

			assert.NotNil(t, qb.GetQuery())
		})
	}
}

func TestQueryBuilder_TextSearch_ShortQuery(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &searchVectorEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := db.NewSelect().
		Model((*searchVectorEntity)(nil)).
		ModelTableExpr("search_vector_entities AS sve")
	qb := NewWithPostgresSearch(query, "sve", fieldConfig, entity)
	qb.ApplyTextSearch("ab", []string{"name"})

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_AllJoinTypes(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}

	tests := []struct {
		name     string
		joinType dbtype.JoinType
	}{
		{"LeftJoin", dbtype.JoinTypeLeft},
		{"RightJoin", dbtype.JoinTypeRight},
		{"InnerJoin", dbtype.JoinTypeInner},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldConfig := &domaintypes.FieldConfiguration{
				FilterableFields: map[string]bool{"related.field": true},
				SortableFields:   map[string]bool{},
				FieldMap:         map[string]string{},
				EnumMap:          map[string]bool{},
				NestedFields: map[string]domaintypes.NestedFieldDefintion{
					"related.field": {
						DatabaseField: "rel.field",
						RequiredJoins: []domaintypes.JoinStep{
							{
								Table:     "related",
								Alias:     "rel",
								Condition: "m.id = rel.m_id",
								JoinType:  tt.joinType,
							},
						},
					},
				},
			}

			filters := []domaintypes.FieldFilter{
				{Field: "related.field", Operator: dbtype.OpEqual, Value: "test"},
			}

			query := newTestQuery(db)
			qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
			qb.ApplyFilters(filters)

			assert.NotNil(t, qb.GetQuery())
		})
	}
}

func TestQueryBuilder_GreaterThanOrEqual(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpGreaterThanOrEqual, Value: 10},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_LessThanOrEqual(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpLessThanOrEqual, Value: 100},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_LikeOperator(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpLike, Value: "%test%"},
		{Field: "code", Operator: dbtype.OpILike, Value: "%ABC%"},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_FieldWithDotInName(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.WithTraversalSupport(false)

	filters := []domaintypes.FieldFilter{
		{Field: "nested.field", Operator: dbtype.OpEqual, Value: "test"},
	}
	qb.ApplyFilters(filters)

	assert.True(t, qb.HasInvalidFields())
	assert.Contains(t, qb.GetInvalidFields(), "nested.field")
}

func TestQueryBuilder_NoTableAlias(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "", fieldConfig, entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
	}
	qb.ApplyFilters(filters)

	sorts := []domaintypes.SortField{
		{Field: "name", Direction: dbtype.SortDirectionAsc},
	}
	qb.ApplySort(sorts)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_FilterGroups_SingleGroup(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
				{Field: "status", Operator: dbtype.OpEqual, Value: "pending"},
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_FilterGroups_MultipleGroups(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
				{Field: "status", Operator: dbtype.OpEqual, Value: "pending"},
			},
		},
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "name", Operator: dbtype.OpContains, Value: "test"},
				{Field: "code", Operator: dbtype.OpStartsWith, Value: "ABC"},
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_FilterGroups_EmptyGroup(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{Filters: []domaintypes.FieldFilter{}},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_FilterGroups_SingleFilter(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_FilterGroups_InvalidField(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "invalid_field", Operator: dbtype.OpEqual, Value: "test"},
				{Field: "name", Operator: dbtype.OpEqual, Value: "valid"},
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.True(t, qb.HasInvalidFields())
	assert.Contains(t, qb.GetInvalidFields(), "invalid_field")
}

func TestQueryBuilder_FilterGroups_WithAllOperators(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
				{Field: "name", Operator: dbtype.OpNotEqual, Value: "other"},
				{Field: "name", Operator: dbtype.OpContains, Value: "abc"},
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_FilterGroups_RelationshipField(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &entityWithRelationship{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "related.name", Operator: dbtype.OpContains, Value: "test"},
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "ewr", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_Sort_RelationshipField(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &entityWithRelationship{}
	fieldConfig := GetFieldConfiguration(entity)

	sorts := []domaintypes.SortField{
		{Field: "related.name", Direction: dbtype.SortDirectionDesc},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "ewr", fieldConfig, entity)
	qb.ApplySort(sorts)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_RelativeDate_LastNDays(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpLastNDays, Value: 7},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_LastNDays_Float(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpLastNDays, Value: float64(30)},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_LastNDays_Map(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpLastNDays, Value: map[string]any{"days": 14}},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_NextNDays(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpNextNDays, Value: 30},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_Today(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpToday, Value: nil},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_Yesterday(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpYesterday, Value: nil},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_Tomorrow(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpTomorrow, Value: nil},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_InvalidValue(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	filters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpLastNDays, Value: "not a number"},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilters(filters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_RelativeDate_InFilterGroup(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	groups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "name", Operator: dbtype.OpToday, Value: nil},
				{Field: "name", Operator: dbtype.OpYesterday, Value: nil},
			},
		},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.ApplyFilterGroups(groups)

	assert.NotNil(t, qb.GetQuery())
}

type geoEntity struct {
	ID   string `json:"id"   bun:"id,pk"`
	Name string `json:"name" bun:"name"`
	Geom string `json:"geom" bun:"geom"`
}

func (g *geoEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ge",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}
}

func (g *geoEntity) GetTableName() string {
	return "geo_entities"
}

func TestQueryBuilder_GeoFilter_WithinRadius(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &geoEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields:    map[string]bool{"name": true, "geom": true},
		SortableFields:      map[string]bool{"name": true},
		GeoFilterableFields: map[string]bool{"geom": true},
		FieldMap:            map[string]string{"name": "name", "geom": "geom"},
		EnumMap:             map[string]bool{},
		NestedFields:        map[string]domaintypes.NestedFieldDefintion{},
	}

	geoFilters := []domaintypes.GeoFilter{
		{
			Field: "geom",
			Center: domaintypes.GeoPoint{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			RadiusKm: 50,
		},
	}

	query := db.NewSelect().Model((*geoEntity)(nil)).ModelTableExpr("geo_entities AS ge")
	qb := NewWithPostgresSearch(query, "ge", fieldConfig, entity)
	qb.ApplyGeoFilters(geoFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_GeoFilter_MultipleFilters(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &geoEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields:    map[string]bool{"name": true, "geom": true, "origin_geom": true},
		SortableFields:      map[string]bool{"name": true},
		GeoFilterableFields: map[string]bool{"geom": true, "origin_geom": true},
		FieldMap: map[string]string{
			"name":        "name",
			"geom":        "geom",
			"origin_geom": "origin_geom",
		},
		EnumMap:      map[string]bool{},
		NestedFields: map[string]domaintypes.NestedFieldDefintion{},
	}

	geoFilters := []domaintypes.GeoFilter{
		{
			Field:    "geom",
			Center:   domaintypes.GeoPoint{Latitude: 40.7128, Longitude: -74.0060},
			RadiusKm: 50,
		},
		{
			Field:    "origin_geom",
			Center:   domaintypes.GeoPoint{Latitude: 34.0522, Longitude: -118.2437},
			RadiusKm: 100,
		},
	}

	query := db.NewSelect().Model((*geoEntity)(nil)).ModelTableExpr("geo_entities AS ge")
	qb := NewWithPostgresSearch(query, "ge", fieldConfig, entity)
	qb.ApplyGeoFilters(geoFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_GeoFilter_ZeroRadius(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &geoEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields:    map[string]bool{"geom": true},
		SortableFields:      map[string]bool{},
		GeoFilterableFields: map[string]bool{"geom": true},
		FieldMap:            map[string]string{"geom": "geom"},
		EnumMap:             map[string]bool{},
		NestedFields:        map[string]domaintypes.NestedFieldDefintion{},
	}

	geoFilters := []domaintypes.GeoFilter{
		{
			Field:    "geom",
			Center:   domaintypes.GeoPoint{Latitude: 40.7128, Longitude: -74.0060},
			RadiusKm: 0,
		},
	}

	query := db.NewSelect().Model((*geoEntity)(nil)).ModelTableExpr("geo_entities AS ge")
	qb := NewWithPostgresSearch(query, "ge", fieldConfig, entity)
	qb.ApplyGeoFilters(geoFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_GeoFilter_LargeRadius(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &geoEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields:    map[string]bool{"geom": true},
		SortableFields:      map[string]bool{},
		GeoFilterableFields: map[string]bool{"geom": true},
		FieldMap:            map[string]string{"geom": "geom"},
		EnumMap:             map[string]bool{},
		NestedFields:        map[string]domaintypes.NestedFieldDefintion{},
	}

	geoFilters := []domaintypes.GeoFilter{
		{
			Field:    "geom",
			Center:   domaintypes.GeoPoint{Latitude: 0, Longitude: 0},
			RadiusKm: 20000,
		},
	}

	query := db.NewSelect().Model((*geoEntity)(nil)).ModelTableExpr("geo_entities AS ge")
	qb := NewWithPostgresSearch(query, "ge", fieldConfig, entity)
	qb.ApplyGeoFilters(geoFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_GeoFilter_InvalidField_IsRejected(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &geoEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields:    map[string]bool{"geom": true},
		SortableFields:      map[string]bool{},
		GeoFilterableFields: map[string]bool{"geom": true},
		FieldMap:            map[string]string{"geom": "geom"},
		EnumMap:             map[string]bool{},
		NestedFields:        map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().Model((*geoEntity)(nil)).ModelTableExpr("geo_entities AS ge")
	qb := NewWithPostgresSearch(query, "ge", fieldConfig, entity)
	qb.ApplyGeoFilters([]domaintypes.GeoFilter{
		{
			Field:    "geom) OR 1=1 --",
			Center:   domaintypes.GeoPoint{Latitude: 40.7128, Longitude: -74.0060},
			RadiusKm: 50,
		},
	})

	assert.True(t, qb.HasInvalidFields())
	assert.Contains(t, qb.GetInvalidFields(), "geom) OR 1=1 --")
	assert.NotContains(t, querySQL(t, qb.GetQuery()), "OR 1=1")
}

type orderEntity struct {
	ID         string `json:"id"         bun:"id,pk"`
	CustomerID string `json:"customerId" bun:"customer_id"`
}

type stopEntity struct {
	ID      string `json:"id"      bun:"id,pk"`
	OrderID string `json:"orderId" bun:"order_id"`
}

func (o *orderEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ord",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "id", Type: domaintypes.FieldTypeText},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "stops",
				Type:         dbtype.RelationshipTypeHasMany,
				TargetEntity: &stopEntity{},
				TargetTable:  "stops",
				ForeignKey:   "order_id",
				ReferenceKey: "id",
				Alias:        "st",
				Queryable:    true,
			},
		},
	}
}

func (o *orderEntity) GetTableName() string {
	return "orders"
}

func TestQueryBuilder_AggregateFilter_CountGt(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{
			Relation: "stops",
			Operator: dbtype.OpCountGt,
			Value:    3,
		},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_AggregateFilter_CountLt(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{
			Relation: "stops",
			Operator: dbtype.OpCountLt,
			Value:    5,
		},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_AggregateFilter_CountEq(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{
			Relation: "stops",
			Operator: dbtype.OpCountEq,
			Value:    2,
		},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_AggregateFilter_CountGte(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{
			Relation: "stops",
			Operator: dbtype.OpCountGte,
			Value:    1,
		},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_AggregateFilter_CountLte(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{
			Relation: "stops",
			Operator: dbtype.OpCountLte,
			Value:    10,
		},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_AggregateFilter_InvalidRelation(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{
			Relation: "nonexistent_relation",
			Operator: dbtype.OpCountGt,
			Value:    1,
		},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.True(t, qb.HasInvalidFields())
	assert.Contains(t, qb.GetInvalidFields(), "nonexistent_relation")
}

func TestQueryBuilder_AggregateFilter_InvalidOperator(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{
			Relation: "stops",
			Operator: dbtype.OpEqual,
			Value:    1,
		},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_AggregateFilter_MultipleFilters(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	aggFilters := []domaintypes.AggregateFilter{
		{Relation: "stops", Operator: dbtype.OpCountGte, Value: 1},
		{Relation: "stops", Operator: dbtype.OpCountLte, Value: 10},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_DistinctValues(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.GetDistinctValues("status")

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_DistinctValues_WithTableAlias(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.GetDistinctValues("name")

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_DistinctValues_NoTableAlias(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "", fieldConfig, entity)
	qb.GetDistinctValues("status")

	assert.NotNil(t, qb.GetQuery())
}

func TestExtractDays_Int(t *testing.T) {
	result := extractDays(7)
	assert.Equal(t, 7, result)
}

func TestExtractDays_Float64(t *testing.T) {
	result := extractDays(float64(30))
	assert.Equal(t, 30, result)
}

func TestExtractDays_Map(t *testing.T) {
	result := extractDays(map[string]any{"days": 14})
	assert.Equal(t, 14, result)
}

func TestExtractDays_MapWithFloatValue(t *testing.T) {
	result := extractDays(map[string]any{"days": float64(21)})
	assert.Equal(t, 21, result)
}

func TestExtractDays_InvalidType(t *testing.T) {
	result := extractDays("invalid")
	assert.Equal(t, 0, result)
}

func TestExtractDays_EmptyMap(t *testing.T) {
	result := extractDays(map[string]any{})
	assert.Equal(t, 0, result)
}

func TestGetDayBounds(t *testing.T) {
	testTime := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)
	start, end := getDayBounds(testTime)

	startTime := time.Unix(start, 0).UTC()
	endTime := time.Unix(end, 0).UTC()

	assert.Equal(t, 2024, startTime.Year())
	assert.Equal(t, time.June, startTime.Month())
	assert.Equal(t, 15, startTime.Day())
	assert.Equal(t, 0, startTime.Hour())
	assert.Equal(t, 0, startTime.Minute())
	assert.Equal(t, 0, startTime.Second())

	assert.Equal(t, 2024, endTime.Year())
	assert.Equal(t, time.June, endTime.Month())
	assert.Equal(t, 15, endTime.Day())
	assert.Equal(t, 23, endTime.Hour())
	assert.Equal(t, 59, endTime.Minute())
	assert.Equal(t, 59, endTime.Second())
}

func TestQueryBuilder_FilterGroups_AllOperators(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	tests := []struct {
		name     string
		operator dbtype.Operator
		value    any
	}{
		{"Equal", dbtype.OpEqual, "test"},
		{"NotEqual", dbtype.OpNotEqual, "test"},
		{"GreaterThan", dbtype.OpGreaterThan, 10},
		{"GreaterThanOrEqual", dbtype.OpGreaterThanOrEqual, 10},
		{"LessThan", dbtype.OpLessThan, 10},
		{"LessThanOrEqual", dbtype.OpLessThanOrEqual, 10},
		{"Contains", dbtype.OpContains, "test"},
		{"StartsWith", dbtype.OpStartsWith, "test"},
		{"EndsWith", dbtype.OpEndsWith, "test"},
		{"Like", dbtype.OpLike, "%test%"},
		{"ILike", dbtype.OpILike, "%test%"},
		{"In", dbtype.OpIn, []string{"a", "b"}},
		{"NotIn", dbtype.OpNotIn, []string{"a", "b"}},
		{"IsNull", dbtype.OpIsNull, nil},
		{"IsNotNull", dbtype.OpIsNotNull, nil},
		{"LastNDays", dbtype.OpLastNDays, 7},
		{"NextNDays", dbtype.OpNextNDays, 30},
		{"Today", dbtype.OpToday, nil},
		{"Yesterday", dbtype.OpYesterday, nil},
		{"Tomorrow", dbtype.OpTomorrow, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := []domaintypes.FilterGroup{
				{
					Filters: []domaintypes.FieldFilter{
						{Field: "name", Operator: tt.operator, Value: tt.value},
					},
				},
			}

			query := newTestQuery(db)
			qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
			qb.ApplyFilterGroups(groups)

			assert.NotNil(t, qb.GetQuery())
		})
	}
}

func TestQueryBuilder_GetDistinctValues_Helper(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: "org_123",
			BuID:  "bu_456",
		},
	}

	query := newTestQuery(db)
	result := GetDistinctValues(query, "m", "status", filter, entity)

	assert.NotNil(t, result)
}

func TestQueryBuilder_GetDistinctValues_InvalidField_IsRejected(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)
	qb.GetDistinctValues("name) desc; drop table users; --")

	assert.True(t, qb.HasInvalidFields())
	assert.Contains(t, qb.GetInvalidFields(), "name) desc; drop table users; --")
	assert.NotContains(t, strings.ToLower(querySQL(t, qb.GetQuery())), "drop table")
}

func TestQueryBuilder_CombinedFilters_AllNewFeatures(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &orderEntity{}
	fieldConfig := GetFieldConfiguration(entity)
	fieldConfig.FilterableFields["geom"] = true
	fieldConfig.GeoFilterableFields["geom"] = true
	fieldConfig.FieldMap["geom"] = "geom"

	fieldFilters := []domaintypes.FieldFilter{
		{Field: "id", Operator: dbtype.OpLastNDays, Value: 30},
	}

	filterGroups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "id", Operator: dbtype.OpToday, Value: nil},
				{Field: "id", Operator: dbtype.OpYesterday, Value: nil},
			},
		},
	}

	geoFilters := []domaintypes.GeoFilter{
		{
			Field:    "geom",
			Center:   domaintypes.GeoPoint{Latitude: 40.7128, Longitude: -74.0060},
			RadiusKm: 100,
		},
	}

	aggFilters := []domaintypes.AggregateFilter{
		{Relation: "stops", Operator: dbtype.OpCountGte, Value: 2},
	}

	query := db.NewSelect().Model((*orderEntity)(nil)).ModelTableExpr("orders AS ord")
	qb := NewWithPostgresSearch(query, "ord", fieldConfig, entity)

	qb.ApplyFilters(fieldFilters)
	qb.ApplyFilterGroups(filterGroups)
	qb.ApplyGeoFilters(geoFilters)
	qb.ApplyAggregateFilters(aggFilters)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_CombinedFiltersAndFilterGroupsWithSort(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &mockEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	fieldFilters := []domaintypes.FieldFilter{
		{Field: "name", Operator: dbtype.OpContains, Value: "test"},
	}

	filterGroups := []domaintypes.FilterGroup{
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
				{Field: "status", Operator: dbtype.OpEqual, Value: "pending"},
			},
		},
		{
			Filters: []domaintypes.FieldFilter{
				{Field: "name", Operator: dbtype.OpStartsWith, Value: "prefix"},
			},
		},
	}

	sorts := []domaintypes.SortField{
		{Field: "name", Direction: dbtype.SortDirectionAsc},
		{Field: "status", Direction: dbtype.SortDirectionDesc},
	}

	query := newTestQuery(db)
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	qb.ApplyFilters(fieldFilters)
	qb.ApplyFilterGroups(filterGroups)
	qb.ApplySort(sorts)

	assert.NotNil(t, qb.GetQuery())
	assert.False(t, qb.HasInvalidFields())
}
