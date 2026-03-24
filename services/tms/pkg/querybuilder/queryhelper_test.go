package querybuilder

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type helperTestEntity struct {
	ID     string `json:"id"     bun:"id,pk"`
	Name   string `json:"name"   bun:"name"`
	Status string `json:"status" bun:"status"`
	Code   string `json:"code"   bun:"code"`
}

func (h *helperTestEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "hte",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "code", Type: domaintypes.FieldTypeText},
		},
	}
}

func (h *helperTestEntity) GetTableName() string {
	return "helper_test_entities"
}

func newHelperTestDB() *bun.DB {
	return bun.NewDB(&sql.DB{}, pgdialect.New())
}

func newHelperTestQuery(db *bun.DB) *bun.SelectQuery {
	return db.NewSelect().
		Model((*helperTestEntity)(nil)).
		ModelTableExpr("helper_test_entities AS hte")
}

func TestApplyFilters(t *testing.T) {
	ClearCaches()

	db := newHelperTestDB()
	entity := &helperTestEntity{}

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
		},
		Sort: []domaintypes.SortField{
			{Field: "name", Direction: dbtype.SortDirectionAsc},
		},
	}

	query := newHelperTestQuery(db)
	result := ApplyFilters(query, "hte", filter, entity)

	assert.NotNil(t, result)
}

func TestApplyFilters_WithQuery(t *testing.T) {
	ClearCaches()

	db := newHelperTestDB()
	entity := &helperTestEntity{}

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		Query: "search term",
	}

	query := newHelperTestQuery(db)
	result := ApplyFilters(query, "hte", filter, entity)

	assert.NotNil(t, result)
}

func TestApplyFilters_EmptyFilter(t *testing.T) {
	ClearCaches()

	db := newHelperTestDB()
	entity := &helperTestEntity{}

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	}

	query := newHelperTestQuery(db)
	result := ApplyFilters(query, "hte", filter, entity)

	assert.NotNil(t, result)
}

func TestApplyFiltersWithConfig(t *testing.T) {
	ClearCaches()

	db := newHelperTestDB()
	entity := &helperTestEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
		},
	}

	query := newHelperTestQuery(db)
	result := ApplyFiltersWithConfig(query, "hte", fieldConfig, filter, entity)

	assert.NotNil(t, result)
}

func TestApplyFiltersWithConfig_CustomConfig(t *testing.T) {
	ClearCaches()

	db := newHelperTestDB()
	entity := &helperTestEntity{}

	customConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true},
		SortableFields:   map[string]bool{"name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "name", Operator: dbtype.OpEqual, Value: "test"},
			{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
		},
	}

	query := newHelperTestQuery(db)
	result := ApplyFiltersWithConfig(query, "hte", customConfig, filter, entity)

	assert.NotNil(t, result)
}

func TestExtractSearchFields(t *testing.T) {
	config := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{
			"name":   true,
			"status": true,
			"code":   true,
		},
		EnumMap: map[string]bool{
			"status": true,
		},
	}

	searchFields := ExtractSearchFields(config)

	assert.Contains(t, searchFields, "name")
	assert.Contains(t, searchFields, "code")
	assert.NotContains(t, searchFields, "status")
}

func TestExtractSearchFields_NoEnums(t *testing.T) {
	config := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{
			"name":        true,
			"description": true,
		},
		EnumMap: map[string]bool{},
	}

	searchFields := ExtractSearchFields(config)

	assert.Len(t, searchFields, 2)
	assert.Contains(t, searchFields, "name")
	assert.Contains(t, searchFields, "description")
}

func TestExtractSearchFields_AllEnums(t *testing.T) {
	config := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{
			"status": true,
			"type":   true,
		},
		EnumMap: map[string]bool{
			"status": true,
			"type":   true,
		},
	}

	searchFields := ExtractSearchFields(config)

	assert.Empty(t, searchFields)
}

func TestExtractSearchFields_EmptyConfig(t *testing.T) {
	config := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		EnumMap:          map[string]bool{},
	}

	searchFields := ExtractSearchFields(config)

	assert.Empty(t, searchFields)
}

func TestExtractSearchFields_NonFilterable(t *testing.T) {
	config := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{
			"name":   true,
			"hidden": false,
		},
		EnumMap: map[string]bool{},
	}

	searchFields := ExtractSearchFields(config)

	assert.Contains(t, searchFields, "name")
	assert.NotContains(t, searchFields, "hidden")
}

func TestApplyFilters_MultipleFilters(t *testing.T) {
	ClearCaches()

	db := newHelperTestDB()
	entity := &helperTestEntity{}

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "name", Operator: dbtype.OpContains, Value: "test"},
			{Field: "status", Operator: dbtype.OpEqual, Value: "active"},
			{Field: "code", Operator: dbtype.OpStartsWith, Value: "ABC"},
		},
		Sort: []domaintypes.SortField{
			{Field: "name", Direction: dbtype.SortDirectionAsc},
			{Field: "status", Direction: dbtype.SortDirectionDesc},
		},
	}

	query := newHelperTestQuery(db)
	result := ApplyFilters(query, "hte", filter, entity)

	assert.NotNil(t, result)
}

func TestApplyFilters_UsesCache(t *testing.T) {
	ClearCaches()

	db := newHelperTestDB()
	entity := &helperTestEntity{}

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	}

	query1 := newHelperTestQuery(db)
	_ = ApplyFilters(query1, "hte", filter, entity)

	config1 := GetFieldConfiguration(entity)

	query2 := newHelperTestQuery(db)
	_ = ApplyFilters(query2, "hte", filter, entity)

	config2 := GetFieldConfiguration(entity)

	assert.Same(t, config1, config2, "Should use cached configuration")
}
