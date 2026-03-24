package domaintypes

import (
	"errors"
	"strings"
)

var ErrInvalidStringOrCommaSeparated = errors.New(
	"invalid format: each comma-separated value must be non-empty",
)

func ValidateStringOrCommaSeparated(value any) error {
	str, ok := value.(string)
	if !ok {
		return ErrInvalidStringOrCommaSeparated
	}

	if str == "" {
		return nil
	}

	for part := range strings.SplitSeq(str, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return ErrInvalidStringOrCommaSeparated
		}
	}

	return nil
}
