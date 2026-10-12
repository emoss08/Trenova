package common

import (
	"fmt"
	"strings"
)

func OptionalScheduleDay(params map[string]any, key string) (*int64, error) {
	raw, given := params[key]
	if !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent day is no day and no error
	}
	if text, isText := raw.(string); !isText || strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("parameter %q must be YYYY-MM-DD", key)
	}
	day, err := RequireUTCDay(params, key)
	if err != nil {
		return nil, err
	}
	seconds := day.Unix()

	return &seconds, nil
}

func RequireScheduleDay(params map[string]any, key string) (int64, error) {
	day, err := RequireUTCDay(params, key)
	if err != nil {
		return 0, err
	}

	return day.Unix(), nil
}
