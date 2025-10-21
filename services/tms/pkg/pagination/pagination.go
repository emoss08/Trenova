package pagination

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

func Params(c *gin.Context) (*Info, error) {
	// Default values
	defaultOffset := 0
	defaultLimit := 10

	offsetStr := c.Query("offset")
	limitStr := c.Query("limit")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = defaultOffset
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = defaultLimit
	}

	return &Info{
		Offset: offset,
		Limit:  limit,
	}, nil
}

func buildPageURL(c *gin.Context, offset, limit int) string {
	req := c.Request
	query := req.URL.Query()
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))

	scheme := "http"
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s?%s", scheme, req.Host, req.URL.Path, query.Encode())
}

func GetNextPageURL(c *gin.Context, limit, offset, totalRows int) string {
	if offset+limit >= totalRows {
		return ""
	}
	return buildPageURL(c, offset+limit, limit)
}

func GetPrevPageURL(c *gin.Context, limit, offset int) string {
	if offset == 0 {
		return ""
	}
	prevOffset := max(offset-limit, 0)
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return buildPageURL(c, prevOffset, limit)
}

func parseFilters( //nolint:gocognit // This is a helper function
	c *gin.Context,
	opts *QueryOptions,
) {
	if filtersJSON := c.Query("filters"); filtersJSON != "" {
		var filters []FieldFilter
		if err := sonic.Unmarshal([]byte(filtersJSON), &filters); err == nil {
			for i := range filters {
				// For IN and NOT_IN operators with array values, keep them as-is
				switch filters[i].Operator { //nolint:exhaustive // We only support the operators we need
				case OpIn, OpNotIn:
					// If the value is already an array, keep it
					if _, isArray := filters[i].Value.([]any); !isArray {
						// If it's not an array, try to parse it
						filters[i].Value = parseFilterValue(
							fmt.Sprintf("%v", filters[i].Value),
							string(filters[i].Operator),
						)
					}
				default:
					// For other operators, parse the value
					filters[i].Value = parseFilterValue(
						fmt.Sprintf("%v", filters[i].Value),
						string(filters[i].Operator),
					)
				}
			}
			opts.FieldFilters = filters
			return
		}
	}

	filtersMap := make(map[int]map[string]string)

	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "filters[") && len(values) > 0 {
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
					filter := FieldFilter{
						Field:    field,
						Operator: FilterOperator(operator),
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
		var sorts []SortField
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

			sort := SortField{
				Field:     field,
				Direction: SortDirection(direction),
			}
			opts.Sort = append(opts.Sort, sort)
		}
	}
}

func parseFilterKey(key string) []string {
	if strings.HasPrefix(key, "filters[") {
		key = key[8:] // Remove "filters["
	} else if strings.HasPrefix(key, "sort[") {
		key = key[5:] // Remove "sort["
	}

	if idx := strings.Index(key, "]["); idx != -1 {
		index := key[:idx]
		field := strings.TrimSuffix(key[idx+2:], "]")
		return []string{index, field}
	}

	return nil
}

func parseFilterValue(value, operator string) any {
	switch FilterOperator(operator) { //nolint:exhaustive // We only support the operators we need
	case OpIn, OpNotIn:
		var arr []any
		if err := sonic.Unmarshal([]byte(value), &arr); err == nil {
			return arr
		}
		return strings.Split(value, ",")
	case OpDateRange:
		var dateRange map[string]any
		if err := sonic.Unmarshal([]byte(value), &dateRange); err == nil {
			return dateRange
		}
		return value
	default:
		return value
	}
}
