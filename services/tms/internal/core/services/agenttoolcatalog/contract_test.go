package agenttoolcatalog_test

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/core/services/agenttoolservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/require"
)

const (
	// maxFirstSentence is what find_tools and the prompt's tool list show of a
	// tool. It has to say what the tool does on its own.
	maxFirstSentence = 160
	// maxDescription bounds what every loaded tool costs a turn.
	maxDescription = 900
)

type describedTool interface {
	Name() string
	Description() string
	ParamSchema() map[string]any
}

// buildTools calls every registered constructor with zero-valued dependencies.
// A constructor only stores what it is given, so the tool it returns can say
// what it is without any of them.
func buildTools(t *testing.T) []describedTool {
	t.Helper()

	// A tool whose schema is built from a catalog needs the catalog; the
	// production one takes no connection.
	supplied := map[reflect.Type]reflect.Value{
		reflect.TypeFor[*filtercatalog.Catalog](): reflect.ValueOf(
			agentquerytoolservice.FilterCatalog(),
		),
	}

	providers := append(agentquerytoolservice.ToolProviders(), agenttoolservice.ToolProviders()...)
	tools := make([]describedTool, 0, len(providers))
	for _, provider := range providers {
		fn := reflect.ValueOf(provider)
		args := make([]reflect.Value, fn.Type().NumIn())
		for idx := range args {
			if value, ok := supplied[fn.Type().In(idx)]; ok {
				args[idx] = value
				continue
			}
			args[idx] = reflect.Zero(fn.Type().In(idx))
		}
		var out []reflect.Value
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("%s panicked when built without dependencies: %v",
						fn.Type(), recovered)
				}
			}()
			out = fn.Call(args)
		}()
		tool, ok := out[0].Interface().(describedTool)
		require.Truef(t, ok, "%s does not build a tool", fn.Type())
		tools = append(tools, tool)
	}

	return tools
}

// idParameter is a parameter whose value has to come from somewhere: an id,
// a key, or a dataset name. A model given one with no word on where it comes
// from writes a plausible-looking value — "on_time_percentage" for a report id.
var idParameter = regexp.MustCompile(`(Id|Ids|Key|Keys)$|^(dataset|reportKey|key)$`)

// selfEvident are id parameters no tool hands out, because they are the
// caller's own choice.
var selfEvident = map[string]struct{}{
	"idempotencyKey": {},
}

// fromContext is an id the turn itself carries rather than a tool: the record
// on the page, the subject a run was started for, or one the person mentioned.
// Naming that source is as good as naming a tool; naming neither is not.
var fromContext = regexp.MustCompile(
	`(?i)\b(the page|the record on screen|this run'?s subject|the run'?s subject|` +
		`the event that started|mentioned record|the thread'?s subject)\b`,
)

/*
Every tool tells a model what it does, when to use it, and where its ids come
from.

The transcripts behind this test failed on selection, not reasoning: a small
model built an aggregate report where a row list was wanted, used a dataset
search as a record search, and invented a report id for a dashboard tile
because nothing on create_dashboard said list_reports hands one out. A
description a model has to guess around is a bug in the description.
*/
func TestEveryToolDescribesItselfWellEnoughToBeChosen(t *testing.T) {
	t.Parallel()

	tools := buildTools(t)
	registered := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		registered[tool.Name()] = struct{}{}
	}

	var problems []string
	for _, tool := range tools {
		name := tool.Name()
		description := tool.Description()

		if first := stringutils.FirstSentence(description); len(first) > maxFirstSentence {
			problems = append(problems, fmt.Sprintf(
				"%s: first sentence is %d characters; find_tools shows only it, so it must "+
					"say what the tool does in %d", name, len(first), maxFirstSentence))
		}
		if len(description) > maxDescription {
			problems = append(problems, fmt.Sprintf(
				"%s: description is %d characters, over %d",
				name,
				len(description),
				maxDescription,
			))
		}

		properties, _ := tool.ParamSchema()["properties"].(map[string]any)
		for param, raw := range properties {
			property, _ := raw.(map[string]any)
			text, _ := property["description"].(string)
			if strings.TrimSpace(text) == "" {
				problems = append(problems, fmt.Sprintf("%s.%s: no description", name, param))
				continue
			}
			if !idParameter.MatchString(param) {
				continue
			}
			if _, ok := selfEvident[param]; ok {
				continue
			}
			if !namesAnotherTool(text, name, registered) && !fromContext.MatchString(text) {
				problems = append(problems, fmt.Sprintf(
					"%s.%s: does not name the tool (or the page or run subject) that supplies it",
					name, param))
			}
		}
	}

	sort.Strings(problems)
	require.Emptyf(t, problems, "%d tools a model cannot choose or call well:\n%s",
		len(problems), strings.Join(problems, "\n"))
}

func namesAnotherTool(text, self string, registered map[string]struct{}) bool {
	for _, word := range regexp.MustCompile(`[a-z]+(?:_[a-z]+)+`).FindAllString(text, -1) {
		if word == self {
			continue
		}
		if _, ok := registered[word]; ok {
			return true
		}
	}

	return false
}

// A family names tools that exist, each in one family, and stays small. A
// misspelt member would never load, and nothing else would say so.
func TestEveryToolFamilyNamesRegisteredTools(t *testing.T) {
	t.Parallel()

	registered := make(map[string]struct{})
	for _, tool := range buildTools(t) {
		registered[tool.Name()] = struct{}{}
	}

	seen := make(map[string]int)
	for idx, family := range agenttoolcatalog.Families() {
		require.GreaterOrEqualf(t, len(family), 2, "family %d has nobody to bring along", idx)
		require.LessOrEqualf(t, len(family), agenttoolcatalog.MaxFamilySize,
			"family %d has %d tools", idx, len(family))
		for _, name := range family {
			_, ok := registered[name]
			require.Truef(t, ok, "family %d names %q, which is not a registered tool", idx, name)
			previous, twice := seen[name]
			require.Falsef(t, twice, "%q is in families %d and %d", name, previous, idx)
			seen[name] = idx
		}
	}
}
