package helpers

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// QueryString retrieves a string value from query parameters.
// Returns the default value if the key doesn't exist or is empty.
func QueryString(c *gin.Context, key string, defaultValue ...string) string {
	value := c.Query(key)
	if value == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return value
}

// QueryStringTrimmed retrieves a trimmed string value from query parameters.
// Removes leading and trailing whitespace.
func QueryStringTrimmed(c *gin.Context, key string, defaultValue ...string) string {
	value := strings.TrimSpace(c.Query(key))
	if value == "" && len(defaultValue) > 0 {
		return strings.TrimSpace(defaultValue[0])
	}
	return value
}

// QueryBool parses the query string value as a boolean.
// Accepts: "true", "false", "1", "0", "yes", "no", "on", "off" (case-insensitive)
// Returns the default value if provided, otherwise returns false.
func QueryBool(c *gin.Context, key string, defaultValue ...bool) bool {
	value := strings.ToLower(strings.TrimSpace(c.Query(key)))

	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}

	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			if len(defaultValue) > 0 {
				return defaultValue[0]
			}
			return false
		}
		return parsed
	}
}

// QueryInt parses the query string value as an integer.
// Returns the default value if parsing fails or if no default is provided, returns 0.
func QueryInt(c *gin.Context, key string, defaultValue ...int) int {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return parsed
}

// QueryInt64 parses the query string value as an int64.
// Returns the default value if parsing fails or if no default is provided, returns 0.
func QueryInt64(c *gin.Context, key string, defaultValue ...int64) int64 {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return parsed
}

// QueryUint parses the query string value as an unsigned integer.
// Returns the default value if parsing fails or if no default is provided, returns 0.
func QueryUint(c *gin.Context, key string, defaultValue ...uint) uint {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return uint(parsed)
}

// QueryUint64 parses the query string value as a uint64.
// Returns the default value if parsing fails or if no default is provided, returns 0.
func QueryUint64(c *gin.Context, key string, defaultValue ...uint64) uint64 {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return parsed
}

// QueryFloat64 parses the query string value as a float64.
// Returns the default value if parsing fails or if no default is provided, returns 0.0.
func QueryFloat64(c *gin.Context, key string, defaultValue ...float64) float64 {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}
	return parsed
}

// QueryTime parses the query string value as a time.Time.
// Supports multiple formats: RFC3339, RFC3339Nano, Unix timestamp (seconds), Unix timestamp (milliseconds)
// Returns the default value if parsing fails or if no default is provided, returns zero time.
func QueryTime(c *gin.Context, key string, defaultValue ...time.Time) time.Time {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return time.Time{}
	}

	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}

	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t
	}

	if timestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
		if timestamp > 1000000000000 {
			return time.Unix(timestamp/1000, (timestamp%1000)*1000000)
		}
		return time.Unix(timestamp, 0)
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return time.Time{}
}

// QueryUnixTime parses the query string value as a Unix timestamp (seconds or milliseconds).
// Returns the timestamp as int64. If the value appears to be in milliseconds (>1000000000000),
// it returns it as-is. Otherwise, it assumes seconds.
// Returns the default value if parsing fails or if no default is provided, returns 0.
func QueryUnixTime(c *gin.Context, key string, defaultValue ...int64) int64 {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return timestamp
}

// QuerySlice returns a slice of strings from query parameters.
// Supports both repeated keys (?key=a&key=b) and comma-separated values (?key=a,b,c)
func QuerySlice(c *gin.Context, key string, defaultValue ...[]string) []string {
	if values, exists := c.GetQueryArray(key); exists && len(values) > 0 {
		var result []string
		for _, v := range values {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	value := strings.TrimSpace(c.Query(key))
	if value != "" {
		parts := strings.Split(value, ",")
		var result []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return []string{}
}

// QueryIntSlice returns a slice of integers from query parameters.
// Supports both repeated keys (?key=1&key=2) and comma-separated values (?key=1,2,3)
func QueryIntSlice(c *gin.Context, key string, defaultValue ...[]int) []int {
	strValues := QuerySlice(c, key)
	if len(strValues) == 0 {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return []int{}
	}

	var result []int
	for _, str := range strValues {
		if val, err := strconv.Atoi(str); err == nil {
			result = append(result, val)
		}
	}

	if len(result) == 0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return result
}

// QueryEnum validates and returns a query parameter value against a set of allowed values.
// Returns the default value if the parameter is not in the allowed set.
func QueryEnum(c *gin.Context, key string, allowed []string, defaultValue ...string) string {
	value := strings.TrimSpace(c.Query(key))

	if slices.Contains(allowed, value) {
		return value
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return ""
}

// QueryEnumIgnoreCase validates and returns a query parameter value against a set of allowed values (case-insensitive).
// Returns the matched allowed value in its original case.
func QueryEnumIgnoreCase(
	c *gin.Context,
	key string,
	allowed []string,
	defaultValue ...string,
) string {
	value := strings.ToLower(strings.TrimSpace(c.Query(key)))

	for _, a := range allowed {
		if strings.EqualFold(a, value) {
			return a
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return ""
}

// QueryBounded returns an integer query parameter value bounded by min and max values.
// If the value is outside the bounds, it's clamped to the nearest bound.
func QueryBounded(c *gin.Context, key string, minValue, maxValue int, defaultValue ...int) int {
	value := QueryInt(c, key, defaultValue...)

	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

// QueryBounded64 returns an int64 query parameter value bounded by min and max values.
// If the value is outside the bounds, it's clamped to the nearest bound.
func QueryBounded64(
	c *gin.Context,
	key string,
	minValue, maxValue int64,
	defaultValue ...int64,
) int64 {
	value := QueryInt64(c, key, defaultValue...)

	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

// HasQuery checks if a query parameter exists, regardless of its value.
func HasQuery(c *gin.Context, key string) bool {
	_, exists := c.GetQuery(key)
	return exists
}

// QueryOrDefault returns the query value if it exists and is non-empty,
// otherwise returns the provided default value.
// This is a simpler alternative to QueryString when you always want a default.
func QueryOrDefault(c *gin.Context, key, defaultValue string) string {
	if value := c.Query(key); value != "" {
		return value
	}
	return defaultValue
}
