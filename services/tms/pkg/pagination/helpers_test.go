package pagination

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func createTestContext(queryParams url.Values) *gin.Context {
	req := httptest.NewRequest(http.MethodGet, "/?"+queryParams.Encode(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

func TestParseFilters_JSONFormat(t *testing.T) {
	params := url.Values{}
	params.Set(
		"fieldFilters",
		`[{"field":"name","operator":"eq","value":"test"},{"field":"status","operator":"in","value":["active","pending"]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 2)
	assert.Equal(t, "name", opts.FieldFilters[0].Field)
	assert.Equal(t, dbtype.OpEqual, opts.FieldFilters[0].Operator)
	assert.Equal(t, "test", opts.FieldFilters[0].Value)

	assert.Equal(t, "status", opts.FieldFilters[1].Field)
	assert.Equal(t, dbtype.OpIn, opts.FieldFilters[1].Operator)
}

func TestParseFilters_ArrayFormat(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters[0][field]", "name")
	params.Set("fieldFilters[0][operator]", "eq")
	params.Set("fieldFilters[0][value]", "test")
	params.Set("fieldFilters[1][field]", "status")
	params.Set("fieldFilters[1][operator]", "contains")
	params.Set("fieldFilters[1][value]", "active")

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 2)
}

func TestParseFilters_Empty(t *testing.T) {
	c := createTestContext(url.Values{})
	opts := &QueryOptions{}

	parseFilters(c, opts)

	assert.Empty(t, opts.FieldFilters)
}

func TestParseFilters_InvalidJSON(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters", `invalid json`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	assert.Empty(t, opts.FieldFilters)
}

func TestParseFilters_InOperator_ArrayValue(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters", `[{"field":"status","operator":"in","value":["a","b","c"]}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpIn, opts.FieldFilters[0].Operator)
	arr, ok := opts.FieldFilters[0].Value.([]string)
	require.True(t, ok, "expected []string after normalization, got %T", opts.FieldFilters[0].Value)
	assert.Len(t, arr, 3)
	assert.Equal(t, []string{"a", "b", "c"}, arr)
}

func TestParseFilters_DateRangeOperator(t *testing.T) {
	params := url.Values{}
	params.Set(
		"fieldFilters",
		`[{"field":"created_at","operator":"daterange","value":{"start":"2024-01-01","end":"2024-12-31"}}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpDateRange, opts.FieldFilters[0].Operator)
	assert.NotNil(t, opts.FieldFilters[0].Value)
}

func TestParseFilterGroups_ValidJSON(t *testing.T) {
	params := url.Values{}
	params.Set(
		"filterGroups",
		`[{"filters":[{"field":"status","operator":"eq","value":"active"},{"field":"status","operator":"eq","value":"pending"}]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 1)
	require.Len(t, opts.FilterGroups[0].Filters, 2)
	assert.Equal(t, "status", opts.FilterGroups[0].Filters[0].Field)
	assert.Equal(t, dbtype.OpEqual, opts.FilterGroups[0].Filters[0].Operator)
	assert.Equal(t, "active", opts.FilterGroups[0].Filters[0].Value)
}

func TestParseFilterGroups_MultipleGroups(t *testing.T) {
	params := url.Values{}
	params.Set(
		"filterGroups",
		`[{"filters":[{"field":"status","operator":"eq","value":"active"}]},{"filters":[{"field":"name","operator":"contains","value":"test"}]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 2)
	assert.Len(t, opts.FilterGroups[0].Filters, 1)
	assert.Len(t, opts.FilterGroups[1].Filters, 1)
}

func TestParseFilterGroups_Empty(t *testing.T) {
	c := createTestContext(url.Values{})
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	assert.Empty(t, opts.FilterGroups)
}

func TestParseFilterGroups_InvalidJSON(t *testing.T) {
	params := url.Values{}
	params.Set("filterGroups", `invalid json`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	assert.Empty(t, opts.FilterGroups)
}

func TestParseFilterGroups_WithInOperator(t *testing.T) {
	params := url.Values{}
	params.Set(
		"filterGroups",
		`[{"filters":[{"field":"status","operator":"in","value":["a","b"]}]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 1)
	require.Len(t, opts.FilterGroups[0].Filters, 1)
	assert.Equal(t, dbtype.OpIn, opts.FilterGroups[0].Filters[0].Operator)
}

func TestParseFilterGroups_WithLastNDays(t *testing.T) {
	params := url.Values{}
	params.Set(
		"filterGroups",
		`[{"filters":[{"field":"created_at","operator":"lastndays","value":"7"}]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 1)
	require.Len(t, opts.FilterGroups[0].Filters, 1)
	assert.Equal(t, dbtype.OpLastNDays, opts.FilterGroups[0].Filters[0].Operator)
	assert.Equal(t, 7, opts.FilterGroups[0].Filters[0].Value)
}

func TestParseFilterGroups_WithToday(t *testing.T) {
	params := url.Values{}
	params.Set(
		"filterGroups",
		`[{"filters":[{"field":"created_at","operator":"today","value":""}]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 1)
	assert.Equal(t, dbtype.OpToday, opts.FilterGroups[0].Filters[0].Operator)
}

func TestParseGeoFilters_ValidJSON(t *testing.T) {
	params := url.Values{}
	params.Set(
		"geoFilters",
		`[{"field":"location","center":{"latitude":40.7128,"longitude":-74.0060},"radiusKm":50}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseGeoFilters(c, opts)

	require.Len(t, opts.GeoFilters, 1)
	assert.Equal(t, "location", opts.GeoFilters[0].Field)
	assert.Equal(t, 40.7128, opts.GeoFilters[0].Center.Latitude)
	assert.Equal(t, -74.0060, opts.GeoFilters[0].Center.Longitude)
	assert.Equal(t, 50.0, opts.GeoFilters[0].RadiusKm)
}

func TestParseGeoFilters_MultipleFilters(t *testing.T) {
	params := url.Values{}
	params.Set(
		"geoFilters",
		`[{"field":"origin","center":{"latitude":40.7128,"longitude":-74.0060},"radiusKm":100},{"field":"destination","center":{"latitude":34.0522,"longitude":-118.2437},"radiusKm":50}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseGeoFilters(c, opts)

	require.Len(t, opts.GeoFilters, 2)
	assert.Equal(t, "origin", opts.GeoFilters[0].Field)
	assert.Equal(t, "destination", opts.GeoFilters[1].Field)
}

func TestParseGeoFilters_Empty(t *testing.T) {
	c := createTestContext(url.Values{})
	opts := &QueryOptions{}

	parseGeoFilters(c, opts)

	assert.Empty(t, opts.GeoFilters)
}

func TestParseGeoFilters_InvalidJSON(t *testing.T) {
	params := url.Values{}
	params.Set("geoFilters", `invalid json`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseGeoFilters(c, opts)

	assert.Empty(t, opts.GeoFilters)
}

func TestParseGeoFilters_ZeroRadius(t *testing.T) {
	params := url.Values{}
	params.Set(
		"geoFilters",
		`[{"field":"location","center":{"latitude":0,"longitude":0},"radiusKm":0}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseGeoFilters(c, opts)

	require.Len(t, opts.GeoFilters, 1)
	assert.Equal(t, 0.0, opts.GeoFilters[0].RadiusKm)
}

func TestParseAggregateFilters_ValidJSON(t *testing.T) {
	params := url.Values{}
	params.Set("aggregateFilters", `[{"relation":"stops","operator":"countgt","value":3}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseAggregateFilters(c, opts)

	require.Len(t, opts.AggregateFilters, 1)
	assert.Equal(t, "stops", opts.AggregateFilters[0].Relation)
	assert.Equal(t, dbtype.OpCountGt, opts.AggregateFilters[0].Operator)
	assert.Equal(t, 3, opts.AggregateFilters[0].Value)
}

func TestParseAggregateFilters_AllOperators(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		expected dbtype.Operator
	}{
		{"CountGt", "countgt", dbtype.OpCountGt},
		{"CountLt", "countlt", dbtype.OpCountLt},
		{"CountEq", "counteq", dbtype.OpCountEq},
		{"CountGte", "countgte", dbtype.OpCountGte},
		{"CountLte", "countlte", dbtype.OpCountLte},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set(
				"aggregateFilters",
				`[{"relation":"items","operator":"`+tt.operator+`","value":5}]`,
			)

			c := createTestContext(params)
			opts := &QueryOptions{}

			parseAggregateFilters(c, opts)

			require.Len(t, opts.AggregateFilters, 1)
			assert.Equal(t, tt.expected, opts.AggregateFilters[0].Operator)
		})
	}
}

func TestParseAggregateFilters_MultipleFilters(t *testing.T) {
	params := url.Values{}
	params.Set(
		"aggregateFilters",
		`[{"relation":"stops","operator":"countgte","value":1},{"relation":"stops","operator":"countlte","value":10}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseAggregateFilters(c, opts)

	require.Len(t, opts.AggregateFilters, 2)
	assert.Equal(t, dbtype.OpCountGte, opts.AggregateFilters[0].Operator)
	assert.Equal(t, dbtype.OpCountLte, opts.AggregateFilters[1].Operator)
}

func TestParseAggregateFilters_Empty(t *testing.T) {
	c := createTestContext(url.Values{})
	opts := &QueryOptions{}

	parseAggregateFilters(c, opts)

	assert.Empty(t, opts.AggregateFilters)
}

func TestParseAggregateFilters_InvalidJSON(t *testing.T) {
	params := url.Values{}
	params.Set("aggregateFilters", `invalid json`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseAggregateFilters(c, opts)

	assert.Empty(t, opts.AggregateFilters)
}

func TestParseSort_JSONFormat(t *testing.T) {
	params := url.Values{}
	params.Set(
		"sort",
		`[{"field":"name","direction":"asc"},{"field":"created_at","direction":"desc"}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseSort(c, opts)

	require.Len(t, opts.Sort, 2)
	assert.Equal(t, "name", opts.Sort[0].Field)
	assert.Equal(t, dbtype.SortDirectionAsc, opts.Sort[0].Direction)
	assert.Equal(t, "created_at", opts.Sort[1].Field)
	assert.Equal(t, dbtype.SortDirectionDesc, opts.Sort[1].Direction)
}

func TestParseSort_ArrayFormat(t *testing.T) {
	params := url.Values{}
	params.Set("sort[0][field]", "name")
	params.Set("sort[0][direction]", "desc")

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseSort(c, opts)

	require.Len(t, opts.Sort, 1)
	assert.Equal(t, "name", opts.Sort[0].Field)
	assert.Equal(t, dbtype.SortDirectionDesc, opts.Sort[0].Direction)
}

func TestParseSort_DefaultDirection(t *testing.T) {
	params := url.Values{}
	params.Set("sort[0][field]", "name")

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseSort(c, opts)

	require.Len(t, opts.Sort, 1)
	assert.Equal(t, dbtype.SortDirectionAsc, opts.Sort[0].Direction)
}

func TestParseSort_Empty(t *testing.T) {
	c := createTestContext(url.Values{})
	opts := &QueryOptions{}

	parseSort(c, opts)

	assert.Empty(t, opts.Sort)
}

func TestParseSort_InvalidJSON(t *testing.T) {
	params := url.Values{}
	params.Set("sort", `invalid json`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseSort(c, opts)

	assert.Empty(t, opts.Sort)
}

func TestParseFilterKey_Filters(t *testing.T) {
	result := parseFilterKey("fieldFilters[0][field]")

	require.Len(t, result, 2)
	assert.Equal(t, "0", result[0])
	assert.Equal(t, "field", result[1])
}

func TestParseFilterKey_Sort(t *testing.T) {
	result := parseFilterKey("sort[1][direction]")

	require.Len(t, result, 2)
	assert.Equal(t, "1", result[0])
	assert.Equal(t, "direction", result[1])
}

func TestParseFilterKey_Invalid(t *testing.T) {
	result := parseFilterKey("invalid_key")

	assert.Nil(t, result)
}

func TestParseFilterValue_InOperator_JSONArray(t *testing.T) {
	result := parseFilterValue(`["a","b","c"]`, "in")

	arr, ok := result.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 3)
}

func TestParseFilterValue_InOperator_CommaSeparated(t *testing.T) {
	result := parseFilterValue("a,b,c", "in")

	arr, ok := result.([]string)
	require.True(t, ok)
	assert.Len(t, arr, 3)
	assert.Equal(t, []string{"a", "b", "c"}, arr)
}

func TestParseFilterValue_DateRange_ValidJSON(t *testing.T) {
	result := parseFilterValue(`{"start":"2024-01-01","end":"2024-12-31"}`, "daterange")

	dateRange, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "2024-01-01", dateRange["start"])
	assert.Equal(t, "2024-12-31", dateRange["end"])
}

func TestParseFilterValue_DateRange_InvalidJSON(t *testing.T) {
	result := parseFilterValue("invalid", "daterange")

	assert.Equal(t, "invalid", result)
}

func TestParseFilterValue_LastNDays_ValidInt(t *testing.T) {
	result := parseFilterValue("7", "lastndays")

	assert.Equal(t, 7, result)
}

func TestParseFilterValue_LastNDays_InvalidInt(t *testing.T) {
	result := parseFilterValue("invalid", "lastndays")

	assert.Equal(t, "invalid", result)
}

func TestParseFilterValue_NextNDays_ValidInt(t *testing.T) {
	result := parseFilterValue("30", "nextndays")

	assert.Equal(t, 30, result)
}

func TestParseFilterValue_Default(t *testing.T) {
	result := parseFilterValue("test", "eq")

	assert.Equal(t, "test", result)
}

func TestCombinedParsing_AllFilterTypes(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters", `[{"field":"name","operator":"eq","value":"test"}]`)
	params.Set(
		"filterGroups",
		`[{"filters":[{"field":"status","operator":"eq","value":"active"}]}]`,
	)
	params.Set(
		"geoFilters",
		`[{"field":"location","center":{"latitude":40.7128,"longitude":-74.0060},"radiusKm":50}]`,
	)
	params.Set("aggregateFilters", `[{"relation":"stops","operator":"countgt","value":2}]`)
	params.Set("sort", `[{"field":"created_at","direction":"desc"}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)
	parseFilterGroups(c, opts)
	parseGeoFilters(c, opts)
	parseAggregateFilters(c, opts)
	parseSort(c, opts)

	assert.Len(t, opts.FieldFilters, 1)
	assert.Len(t, opts.FilterGroups, 1)
	assert.Len(t, opts.GeoFilters, 1)
	assert.Len(t, opts.AggregateFilters, 1)
	assert.Len(t, opts.Sort, 1)
}

func TestGetNextPageURL(t *testing.T) {
	params := url.Values{}
	req := httptest.NewRequest(http.MethodGet, "/api/items?"+params.Encode(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetNextPageURL(c, 10, 0, 100)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "offset=10")
}

func TestGetNextPageURL_NoMorePages(t *testing.T) {
	params := url.Values{}
	req := httptest.NewRequest(http.MethodGet, "/api/items?"+params.Encode(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetNextPageURL(c, 10, 90, 100)

	assert.Empty(t, result)
}

func TestGetPreviousPageURL(t *testing.T) {
	params := url.Values{}
	req := httptest.NewRequest(http.MethodGet, "/api/items?"+params.Encode(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetPreviousPageURL(c, 10, 20)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "offset=10")
}

func TestGetPreviousPageURL_FirstPage(t *testing.T) {
	params := url.Values{}
	req := httptest.NewRequest(http.MethodGet, "/api/items?"+params.Encode(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetPreviousPageURL(c, 10, 0)

	assert.Empty(t, result)
}

func TestParseFilters_NotInOperator(t *testing.T) {
	params := url.Values{}
	params.Set(
		"fieldFilters",
		`[{"field":"status","operator":"notin","value":["deleted","archived"]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpNotIn, opts.FieldFilters[0].Operator)
}

func TestParseFilterGroups_EmptyFiltersArray(t *testing.T) {
	params := url.Values{}
	params.Set("filterGroups", `[{"filters":[]}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 1)
	assert.Empty(t, opts.FilterGroups[0].Filters)
}

func TestParseGeoFilters_NegativeCoordinates(t *testing.T) {
	params := url.Values{}
	params.Set(
		"geoFilters",
		`[{"field":"location","center":{"latitude":-33.8688,"longitude":151.2093},"radiusKm":25}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseGeoFilters(c, opts)

	require.Len(t, opts.GeoFilters, 1)
	assert.Equal(t, -33.8688, opts.GeoFilters[0].Center.Latitude)
	assert.Equal(t, 151.2093, opts.GeoFilters[0].Center.Longitude)
}

func TestParseAggregateFilters_ZeroValue(t *testing.T) {
	params := url.Values{}
	params.Set("aggregateFilters", `[{"relation":"items","operator":"counteq","value":0}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseAggregateFilters(c, opts)

	require.Len(t, opts.AggregateFilters, 1)
	assert.Equal(t, 0, opts.AggregateFilters[0].Value)
}

func TestParseFilterGroups_WithInOperator_StringValue(t *testing.T) {
	params := url.Values{}
	params.Set("filterGroups", `[{"filters":[{"field":"status","operator":"in","value":"a,b,c"}]}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 1)
	require.Len(t, opts.FilterGroups[0].Filters, 1)
	assert.Equal(t, dbtype.OpIn, opts.FilterGroups[0].Filters[0].Operator)
}

func TestParseFilterGroups_WithNotInOperator_ArrayValue(t *testing.T) {
	params := url.Values{}
	params.Set(
		"filterGroups",
		`[{"filters":[{"field":"status","operator":"notin","value":["x","y"]}]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilterGroups(c, opts)

	require.Len(t, opts.FilterGroups, 1)
	require.Len(t, opts.FilterGroups[0].Filters, 1)
	assert.Equal(t, dbtype.OpNotIn, opts.FilterGroups[0].Filters[0].Operator)
}

func TestParseFilters_MissingFieldOrOperator(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters[0][field]", "name")

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	assert.Empty(t, opts.FieldFilters)
}

func TestParseFilters_MissingValue(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters[0][field]", "name")
	params.Set("fieldFilters[0][operator]", "eq")

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	assert.Empty(t, opts.FieldFilters)
}

func TestParseFilters_InOperator_StringValueInJSON(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters", `[{"field":"status","operator":"in","value":"a,b,c"}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpIn, opts.FieldFilters[0].Operator)
}

func TestParseFilters_NotInOperator_StringValueInJSON(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters", `[{"field":"status","operator":"notin","value":"x,y,z"}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpNotIn, opts.FieldFilters[0].Operator)
}

func TestNormalizeSlice_StringSlice(t *testing.T) {
	input := []any{"Active", "Inactive", "Draft"}
	result := normalizeSlice(input)

	strSlice, ok := result.([]string)
	require.True(t, ok, "expected []string, got %T", result)
	assert.Equal(t, []string{"Active", "Inactive", "Draft"}, strSlice)
}

func TestNormalizeSlice_NumberSlice(t *testing.T) {
	input := []any{float64(1), float64(2), float64(3)}
	result := normalizeSlice(input)

	intSlice, ok := result.([]int64)
	require.True(t, ok, "expected []int64, got %T", result)
	assert.Equal(t, []int64{1, 2, 3}, intSlice)
}

func TestNormalizeSlice_EmptySlice(t *testing.T) {
	input := []any{}
	result := normalizeSlice(input)

	strSlice, ok := result.([]string)
	require.True(t, ok, "expected []string for empty slice, got %T", result)
	assert.Empty(t, strSlice)
}

func TestNormalizeSlice_MixedTypes(t *testing.T) {
	input := []any{"string", float64(123)}
	result := normalizeSlice(input)

	_, ok := result.([]any)
	assert.True(t, ok, "expected []any for mixed types, got %T", result)
}

func TestParseFilters_InOperator_ReturnsTypedSlice(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters", `[{"field":"status","operator":"in","value":["Active","Inactive"]}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpIn, opts.FieldFilters[0].Operator)

	strSlice, ok := opts.FieldFilters[0].Value.([]string)
	require.True(t, ok, "expected []string after normalization, got %T", opts.FieldFilters[0].Value)
	assert.Equal(t, []string{"Active", "Inactive"}, strSlice)
}

func TestParseFilters_NotInOperator_ReturnsTypedSlice(t *testing.T) {
	params := url.Values{}
	params.Set("fieldFilters", `[{"field":"id","operator":"notin","value":[1,2,3]}]`)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpNotIn, opts.FieldFilters[0].Operator)

	intSlice, ok := opts.FieldFilters[0].Value.([]int64)
	require.True(t, ok, "expected []int64 after normalization, got %T", opts.FieldFilters[0].Value)
	assert.Equal(t, []int64{1, 2, 3}, intSlice)
}

func TestNormalizeFilterValue_ArrayOfStrings(t *testing.T) {
	input := []any{"a", "b", "c"}
	result := normalizeFilterValue(input, "in")

	strSlice, ok := result.([]string)
	require.True(t, ok, "expected []string, got %T", result)
	assert.Equal(t, []string{"a", "b", "c"}, strSlice)
}

func TestNormalizeFilterValue_ArrayOfNumbers(t *testing.T) {
	input := []any{float64(10), float64(20), float64(30)}
	result := normalizeFilterValue(input, "in")

	intSlice, ok := result.([]int64)
	require.True(t, ok, "expected []int64, got %T", result)
	assert.Equal(t, []int64{10, 20, 30}, intSlice)
}

func TestNormalizeSlice_WithNilValues(t *testing.T) {
	input := []any{"Active", "Inactive", nil}
	result := normalizeSlice(input)

	strSlice, ok := result.([]string)
	require.True(t, ok, "expected []string, got %T", result)
	assert.Equal(t, []string{"Active", "Inactive"}, strSlice)
}

func TestNormalizeSlice_AllNilValues(t *testing.T) {
	input := []any{nil, nil, nil}
	result := normalizeSlice(input)

	strSlice, ok := result.([]string)
	require.True(t, ok, "expected []string for all-nil slice, got %T", result)
	assert.Empty(t, strSlice)
}

func TestParseFilters_InOperator_WithNullInArray(t *testing.T) {
	params := url.Values{}
	params.Set(
		"fieldFilters",
		`[{"field":"status","operator":"in","value":["Active","Inactive",null]}]`,
	)

	c := createTestContext(params)
	opts := &QueryOptions{}

	parseFilters(c, opts)

	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, dbtype.OpIn, opts.FieldFilters[0].Operator)

	strSlice, ok := opts.FieldFilters[0].Value.([]string)
	require.True(t, ok, "expected []string after normalization, got %T", opts.FieldFilters[0].Value)
	assert.Equal(t, []string{"Active", "Inactive"}, strSlice)
}
