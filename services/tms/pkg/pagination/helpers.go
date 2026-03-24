package pagination

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/gin-gonic/gin"
)

func NewQueryOptions(c *gin.Context, authCtx *authctx.AuthContext) *QueryOptions {
	opts := new(QueryOptions)
	opts.TenantInfo = TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	opts.Pagination = Info{
		Limit:  DefaultLimit,
		Offset: DefaultOffset,
	}

	opts.Query = helpers.QueryString(c, "query", "")

	_ = c.ShouldBindQuery(opts)

	return opts
}

func NewSelectQueryRequest(c *gin.Context, authCtx *authctx.AuthContext) *SelectQueryRequest {
	req := new(SelectQueryRequest)
	req.TenantInfo = TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	req.Pagination = Info{
		Limit:  DefaultLimit,
		Offset: DefaultOffset,
	}
	req.Query = helpers.QueryString(c, "query", "")

	_ = c.ShouldBindQuery(req)

	return req
}

func List[T any](
	c *gin.Context,
	opts *QueryOptions,
	eh *helpers.ErrorHandler,
	fn func() (*ListResult[T], error),
) {
	if err := c.ShouldBindQuery(opts); err != nil {
		eh.HandleError(c, err)
		return
	}

	parseFilters(c, opts)
	parseFilterGroups(c, opts)
	parseGeoFilters(c, opts)
	parseAggregateFilters(c, opts)
	parseSort(c, opts)

	result, err := fn()
	if err != nil {
		eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, Response[[]T]{
		Count:   result.Total,
		Results: result.Items,
		Next:    GetNextPageURL(c, opts.Pagination.Limit, opts.Pagination.Offset, result.Total),
		Prev:    GetPreviousPageURL(c, opts.Pagination.Limit, opts.Pagination.Offset),
	})
}

func SelectOptions[T any](
	c *gin.Context,
	req *SelectQueryRequest,
	eh *helpers.ErrorHandler,
	fn func() (*ListResult[T], error),
) {
	result, err := fn()
	if err != nil {
		eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, Response[[]T]{
		Count:   result.Total,
		Results: result.Items,
		Next:    GetNextPageURL(c, req.Pagination.Limit, req.Pagination.Offset, result.Total),
		Prev:    GetPreviousPageURL(c, req.Pagination.Limit, req.Pagination.Offset),
	})
}

func parseFilters(
	c *gin.Context,
	opts *QueryOptions,
) {
	if filtersJSON := c.Query("fieldFilters"); filtersJSON != "" {
		var filters []domaintypes.FieldFilter
		if err := sonic.Unmarshal([]byte(filtersJSON), &filters); err == nil {
			for i := range filters {
				filters[i].Value = normalizeFilterValue(
					filters[i].Value,
					string(filters[i].Operator),
				)
			}
			opts.FieldFilters = filters
			return
		}
	}

	filtersMap := make(map[int]map[string]string)

	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "fieldFilters[") && len(values) > 0 {
			if parts := parseFilterKey(key); len(parts) == 2 {
				index, _ := strconv.Atoi(parts[0])
				field := parts[1]

				if filtersMap[index] == nil {
					filtersMap[index] = make(map[string]string)
				}
				filtersMap[index][field] = values[0]
			}
		}
	}

	for _, filterMap := range filtersMap {
		if field, hasField := filterMap["field"]; hasField {
			if operator, hasOperator := filterMap["operator"]; hasOperator {
				if value, hasValue := filterMap["value"]; hasValue {
					filter := domaintypes.FieldFilter{
						Field:    field,
						Operator: dbtype.Operator(operator),
						Value:    parseFilterValue(value, operator),
					}
					opts.FieldFilters = append(opts.FieldFilters, filter)
				}
			}
		}
	}
}

func parseSort(c *gin.Context, opts *QueryOptions) {
	if sortJSON := c.Query("sort"); sortJSON != "" {
		var sorts []domaintypes.SortField
		if err := sonic.Unmarshal([]byte(sortJSON), &sorts); err == nil {
			opts.Sort = sorts
			return
		}
	}

	sortMap := make(map[int]map[string]string)

	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "sort[") && len(values) > 0 {
			if parts := parseFilterKey(key); len(parts) == 2 {
				index, _ := strconv.Atoi(parts[0])
				field := parts[1]

				if sortMap[index] == nil {
					sortMap[index] = make(map[string]string)
				}
				sortMap[index][field] = values[0]
			}
		}
	}

	for _, sortEntry := range sortMap {
		if field, hasField := sortEntry["field"]; hasField {
			direction := sortEntry["direction"]
			if direction == "" {
				direction = "asc"
			}

			sort := domaintypes.SortField{
				Field:     field,
				Direction: dbtype.SortDirection(direction),
			}
			opts.Sort = append(opts.Sort, sort)
		}
	}
}

func parseFilterKey(key string) []string {
	if strings.HasPrefix(key, "fieldFilters[") {
		key = key[13:] // Remove "fieldFilters["
	} else if strings.HasPrefix(key, "sort[") {
		key = key[5:] // Remove "sort["
	}

	if before, after, ok := strings.Cut(key, "]["); ok {
		index := before
		field := strings.TrimSuffix(after, "]")
		return []string{index, field}
	}

	return nil
}

func parseFilterValue(value, operator string) any {
	switch dbtype.Operator(operator) { //nolint:exhaustive // We only support the operators we need
	case dbtype.OpIn, dbtype.OpNotIn:
		var arr []any
		if err := sonic.Unmarshal([]byte(value), &arr); err == nil {
			return arr
		}
		return strings.Split(value, ",")
	case dbtype.OpDateRange:
		var dateRange map[string]any
		if err := sonic.Unmarshal([]byte(value), &dateRange); err == nil {
			return dateRange
		}
		return value
	case dbtype.OpLastNDays, dbtype.OpNextNDays:
		if days, err := strconv.Atoi(value); err == nil {
			return days
		}
		return value
	default:
		return value
	}
}

func normalizeFilterValue(value any, operator string) any {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			return int64(v)
		}
		return v
	case []any:
		return normalizeSlice(v)
	case map[string]any:
		return v
	case string:
		return parseFilterValue(v, operator)
	default:
		return value
	}
}

func normalizeSlice(slice []any) any {
	filtered := make([]any, 0, len(slice))
	for _, elem := range slice {
		if elem != nil {
			filtered = append(filtered, elem)
		}
	}

	if len(filtered) == 0 {
		return []string{}
	}

	allStrings := true
	allNumbers := true

	for _, elem := range filtered {
		switch elem.(type) {
		case string:
			allNumbers = false
		case float64:
			allStrings = false
		default:
			allStrings = false
			allNumbers = false
		}
	}

	if allStrings {
		result := make([]string, len(filtered))
		for i, elem := range filtered {
			result[i] = elem.(string) //nolint:errcheck // We know the type is string
		}
		return result
	}

	if allNumbers {
		result := make([]int64, len(filtered))
		for i, elem := range filtered {
			result[i] = int64(elem.(float64)) //nolint:errcheck // We know the type is float64
		}
		return result
	}

	return filtered
}

func parseFilterGroups(c *gin.Context, opts *QueryOptions) {
	groupsJSON := c.Query("filterGroups")
	if groupsJSON == "" {
		return
	}

	var groups []domaintypes.FilterGroup
	if err := sonic.Unmarshal([]byte(groupsJSON), &groups); err != nil {
		return
	}

	for i := range groups {
		for j := range groups[i].Filters {
			filter := &groups[i].Filters[j]
			filter.Value = normalizeFilterValue(filter.Value, string(filter.Operator))
		}
	}

	opts.FilterGroups = groups
}

func parseGeoFilters(c *gin.Context, opts *QueryOptions) {
	geoJSON := c.Query("geoFilters")
	if geoJSON == "" {
		return
	}

	var geoFilters []domaintypes.GeoFilter
	if err := sonic.Unmarshal([]byte(geoJSON), &geoFilters); err != nil {
		return
	}

	opts.GeoFilters = geoFilters
}

func parseAggregateFilters(c *gin.Context, opts *QueryOptions) {
	aggJSON := c.Query("aggregateFilters")
	if aggJSON == "" {
		return
	}

	var aggFilters []domaintypes.AggregateFilter
	if err := sonic.Unmarshal([]byte(aggJSON), &aggFilters); err != nil {
		return
	}

	opts.AggregateFilters = aggFilters
}
