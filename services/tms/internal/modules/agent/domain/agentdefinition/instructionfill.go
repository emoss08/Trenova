package agentdefinition

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	fallbackOrganization = "your organization"
	fallbackUserName     = "the person asking"
	fallbackUserRole     = "their role"
	fallbackToday        = "today"
)

func FillInstructionVariables(text string, rc *RuntimeContext) string {
	if !strings.Contains(text, "{{") {
		return text
	}

	return instructionVariablePattern.ReplaceAllStringFunc(text, func(match string) string {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
		switch name {
		case "organization":
			return stringutils.FirstNonEmptyTrimmed(rc.OrganizationName, fallbackOrganization)
		case "user.name":
			if rc.User == nil {
				return fallbackUserName
			}
			return stringutils.FirstNonEmptyTrimmed(rc.User.Name, fallbackUserName)
		case "user.role":
			if rc.User == nil || len(rc.User.Roles) == 0 {
				return fallbackUserRole
			}
			return strings.Join(rc.User.Roles, ", ")
		case "today":
			return formatToday(rc.Now, rc.Timezone)
		default:
			return match
		}
	})
}

func formatToday(now int64, timezone string) string {
	if now == 0 {
		return fallbackToday
	}
	loc := time.UTC
	if trimmed := strings.TrimSpace(timezone); trimmed != "" {
		if loaded, err := time.LoadLocation(trimmed); err == nil {
			loc = loaded
		}
	}

	return time.Unix(now, 0).In(loc).Format("Monday, January 2, 2006")
}
