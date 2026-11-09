package helpers

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
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

func ContextValue[T any](c *gin.Context, key string) (T, bool) {
	val, exists := c.Get(key)
	if !exists {
		var zero T
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}

func ContextValueOr[T any](c *gin.Context, key string, fallback T) T {
	value, ok := ContextValue[T](c, key)
	if !ok {
		return fallback
	}
	return value
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

func QueryBounded(c *gin.Context, key string, minValue, maxValue int, defaultValue ...int) int {
	return intutils.Clamp(QueryInt(c, key, defaultValue...), minValue, maxValue)
}
