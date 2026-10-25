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

func filterArgs(filter *aianalyticsports.Filter) sqlArgs {
	args := tenantArgs(filter.TenantInfo)
	args.Since = filter.Since
	args.SinceHour = ceilHour(filter.Since)
	args.Agent = filter.AgentDefinitionID.String()
	args.Task = string(filter.Task)

	return args
}

const (
	castText   = "text"
	castBigint = "bigint"
)

type sqlParam struct {
	cast  string
	value func(*sqlArgs) any
}

var sqlParams = map[string]sqlParam{
	"org":           {cast: castText, value: func(a *sqlArgs) any { return a.Org }},
	"bu":            {cast: castText, value: func(a *sqlArgs) any { return a.Bu }},
	"agent":         {cast: castText, value: func(a *sqlArgs) any { return a.Agent }},
	"task":          {cast: castText, value: func(a *sqlArgs) any { return a.Task }},
	"series_key":    {cast: castText, value: func(a *sqlArgs) any { return a.SeriesKey }},
	"tool_name":     {cast: castText, value: func(a *sqlArgs) any { return a.ToolName }},
	"from":          {cast: castBigint, value: func(a *sqlArgs) any { return a.From }},
	"to":            {cast: castBigint, value: func(a *sqlArgs) any { return a.To }},
	"from_hour":     {cast: castBigint, value: func(a *sqlArgs) any { return a.FromHour }},
	"to_hour":       {cast: castBigint, value: func(a *sqlArgs) any { return a.ToHour }},
	"stamp":         {cast: castBigint, value: func(a *sqlArgs) any { return a.Stamp }},
	"since":         {cast: castBigint, value: func(a *sqlArgs) any { return a.Since }},
	"since_hour":    {cast: castBigint, value: func(a *sqlArgs) any { return a.SinceHour }},
	"pending_since": {cast: castBigint, value: func(a *sqlArgs) any { return a.Pending }},
	"before":        {cast: castBigint, value: func(a *sqlArgs) any { return a.Before }},
	"before_hour":   {cast: castBigint, value: func(a *sqlArgs) any { return a.BeforeHour }},
	"limit":         {cast: castBigint, value: func(a *sqlArgs) any { return int64(a.Limit) }},
	"candidates":    {cast: castBigint, value: func(a *sqlArgs) any { return int64(a.Candidates) }},
	"starts":        {cast: "bigint[]", value: func(a *sqlArgs) any { return a.Starts }},
}

func (a *sqlArgs) lookup(name string) (value any, cast string, ok bool) {
	param, known := sqlParams[name]
	if !known {
		return nil, "", false
	}

	return param.value(a), param.cast, true
}

func isNameByte(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z')
}

func bind(query string, args *sqlArgs) (sql string, bound []any, err error) {
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
