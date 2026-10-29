package agentdefinition

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/cronutils"
)

func CronExpressionValid(expression string) bool {
	trimmed := strings.TrimSpace(expression)

	return trimmed != "" && len(trimmed) <= MaxCronLength && cronutils.Validate(trimmed) == nil
}

func CronTimezoneValid(timezone string) bool {
	trimmed := strings.TrimSpace(timezone)
	if trimmed == "" {
		return false
	}
	_, err := time.LoadLocation(trimmed)

	return err == nil
}
