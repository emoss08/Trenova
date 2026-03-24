package helpers

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
)

func queryValue[T any](
	c *gin.Context,
	key string,
	parse func(string) (T, bool),
	zero T,
	defaults []T,
) T {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return sliceutils.FirstOrDefault(defaults, zero)
	}
	if result, ok := parse(value); ok {
		return result
	}
	return sliceutils.FirstOrDefault(defaults, zero)
}

func QueryString(c *gin.Context, key string, defaultValue ...string) string {
	return queryValue(c, key, func(s string) (string, bool) { return s, true }, "", defaultValue)
}

func QueryStringTrimmed(c *gin.Context, key string, defaultValue ...string) string {
	return queryValue(
		c,
		key,
		func(s string) (string, bool) { return strings.TrimSpace(s), true },
		"",
		defaultValue,
	)
}

func QueryBool(c *gin.Context, key string, defaultValue ...bool) bool {
	return queryValue(c, key, stringutils.ParseBool, false, defaultValue)
}

func QueryInt(c *gin.Context, key string, defaultValue ...int) int {
	return queryValue(c, key, func(s string) (int, bool) {
		v, err := strconv.Atoi(s)
		return v, err == nil
	}, 0, defaultValue)
}

func QueryInt64(c *gin.Context, key string, defaultValue ...int64) int64 {
	return queryValue(c, key, func(s string) (int64, bool) {
		v, err := strconv.ParseInt(s, 10, 64)
		return v, err == nil
	}, 0, defaultValue)
}

func QueryUint(c *gin.Context, key string, defaultValue ...uint) uint {
	return queryValue(c, key, func(s string) (uint, bool) {
		v, err := strconv.ParseUint(s, 10, 64)
		return uint(v), err == nil
	}, 0, defaultValue)
}

func QueryUint64(c *gin.Context, key string, defaultValue ...uint64) uint64 {
	return queryValue(c, key, func(s string) (uint64, bool) {
		v, err := strconv.ParseUint(s, 10, 64)
		return v, err == nil
	}, 0, defaultValue)
}

func QueryFloat64(c *gin.Context, key string, defaultValue ...float64) float64 {
	return queryValue(c, key, func(s string) (float64, bool) {
		v, err := strconv.ParseFloat(s, 64)
		return v, err == nil
	}, 0.0, defaultValue)
}

func QueryTime(c *gin.Context, key string, defaultValue ...time.Time) time.Time {
	return queryValue(c, key, timeutils.ParseTimeRFC3339, time.Time{}, defaultValue)
}

func QueryUnixTime(c *gin.Context, key string, defaultValue ...int64) int64 {
	return queryValue(c, key, func(s string) (int64, bool) {
		v, err := strconv.ParseInt(s, 10, 64)
		return v, err == nil
	}, 0, defaultValue)
}

func QuerySlice(c *gin.Context, key string, defaultValue ...[]string) []string {
	values, exists := c.GetQueryArray(key)
	if !exists || len(values) == 0 {
		return sliceutils.FirstOrDefault(defaultValue, []string{})
	}

	if len(values) == 1 && strings.Contains(values[0], ",") {
		if result := stringutils.FilterEmpty(strings.Split(values[0], ",")); len(result) > 0 {
			return result
		}
	}

	if result := stringutils.FilterEmpty(values); len(result) > 0 {
		return result
	}

	return sliceutils.FirstOrDefault(defaultValue, []string{})
}

func QueryIntSlice(c *gin.Context, key string, defaultValue ...[]int) []int {
	strValues := QuerySlice(c, key)
	if len(strValues) == 0 {
		return sliceutils.FirstOrDefault(defaultValue, []int{})
	}

	var result []int
	for _, str := range strValues {
		if val, err := strconv.Atoi(str); err == nil {
			result = append(result, val)
		}
	}

	if len(result) == 0 {
		return sliceutils.FirstOrDefault(defaultValue, []int{})
	}
	return result
}

func QueryEnum(c *gin.Context, key string, allowed []string, defaultValue ...string) string {
	value := strings.TrimSpace(c.Query(key))
	if slices.Contains(allowed, value) {
		return value
	}
	return sliceutils.FirstOrDefault(defaultValue, "")
}

func QueryEnumIgnoreCase(
	c *gin.Context,
	key string,
	allowed []string,
	defaultValue ...string,
) string {
	value := strings.TrimSpace(c.Query(key))
	for _, a := range allowed {
		if strings.EqualFold(a, value) {
			return a
		}
	}

	return sliceutils.FirstOrDefault(defaultValue, "")
}

func QueryBounded(c *gin.Context, key string, minValue, maxValue int, defaultValue ...int) int {
	return intutils.Clamp(QueryInt(c, key, defaultValue...), minValue, maxValue)
}

func QueryBounded64(
	c *gin.Context,
	key string,
	minValue, maxValue int64,
	defaultValue ...int64,
) int64 {
	return intutils.Clamp(QueryInt64(c, key, defaultValue...), minValue, maxValue)
}

func HasQuery(c *gin.Context, key string) bool {
	_, exists := c.GetQuery(key)
	return exists
}

func QueryOrDefault(c *gin.Context, key, defaultValue string) string {
	if value := c.Query(key); value != "" {
		return value
	}
	return defaultValue
}

func QueryPulid(c *gin.Context, key string) pulid.ID {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return pulid.Nil
	}
	id, err := pulid.MustParse(value)
	if err != nil {
		return pulid.Nil
	}
	return id
}
