package aianalyticsrepository

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type sqlArgs struct {
	Org        string
	Bu         string
	From       int64
	To         int64
	FromHour   int64
	ToHour     int64
	Stamp      int64
	Since      int64
	SinceHour  int64
	Agent      string
	Task       string
	Starts     []int64
	SeriesKey  string
	ToolName   string
	Limit      int
	Candidates int
	Pending    int64
	Before     int64
	BeforeHour int64
}

func tenantArgs(tenant pagination.TenantInfo) sqlArgs {
	return sqlArgs{Org: tenant.OrgID.String(), Bu: tenant.BuID.String()}
}

func filterArgs(filter aianalyticsports.Filter) sqlArgs {
	args := tenantArgs(filter.TenantInfo)
	args.Since = filter.Since
	args.SinceHour = ceilHour(filter.Since)
	args.Agent = filter.AgentDefinitionID.String()
	args.Task = string(filter.Task)

	return args
}

func (a *sqlArgs) lookup(name string) (value any, cast string, ok bool) {
	switch name {
	case "org":
		return a.Org, "text", true
	case "bu":
		return a.Bu, "text", true
	case "agent":
		return a.Agent, "text", true
	case "task":
		return a.Task, "text", true
	case "series_key":
		return a.SeriesKey, "text", true
	case "tool_name":
		return a.ToolName, "text", true
	case "from":
		return a.From, "bigint", true
	case "to":
		return a.To, "bigint", true
	case "from_hour":
		return a.FromHour, "bigint", true
	case "to_hour":
		return a.ToHour, "bigint", true
	case "stamp":
		return a.Stamp, "bigint", true
	case "since":
		return a.Since, "bigint", true
	case "since_hour":
		return a.SinceHour, "bigint", true
	case "pending_since":
		return a.Pending, "bigint", true
	case "before":
		return a.Before, "bigint", true
	case "before_hour":
		return a.BeforeHour, "bigint", true
	case "limit":
		return int64(a.Limit), "bigint", true
	case "candidates":
		return int64(a.Candidates), "bigint", true
	case "starts":
		return a.Starts, "bigint[]", true
	default:
		return nil, "", false
	}
}

func isNameByte(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z')
}

func bind(query string, args *sqlArgs) (string, []any, error) {
	var out strings.Builder
	out.Grow(len(query) + len(query)/8)
	positions := make(map[string]int, 8)
	values := make([]any, 0, 8)
	quoted := false
	for index := 0; index < len(query); index++ {
		c := query[index]
		if c == '\'' {
			quoted = !quoted
			out.WriteByte(c)
			continue
		}
		if quoted || c != '?' {
			out.WriteByte(c)
			continue
		}
		end := index + 1
		for end < len(query) && isNameByte(query[end]) {
			end++
		}
		name := query[index+1 : end]
		value, cast, ok := args.lookup(name)
		if !ok {
			return "", nil, fmt.Errorf("unknown analytics parameter %q", name)
		}
		position, seen := positions[name]
		if !seen {
			values = append(values, value)
			position = len(values)
			positions[name] = position
		}
		out.WriteByte('$')
		out.WriteString(strconv.Itoa(position))
		out.WriteString("::")
		out.WriteString(cast)
		index = end - 1
	}

	return out.String(), values, nil
}
