package agenttoolcatalog_test

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/core/services/agenttoolservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/toolschema"
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
	"taxId":          {},
	"externalId":     {},
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
		problems = append(problems, parameterProblems(parameterCheck{
			tool:       name,
			registered: registered,
		}, properties, "", "")...)
	}

	sort.Strings(problems)
	require.Emptyf(t, problems, "%d tools a model cannot choose or call well:\n%s",
		len(problems), strings.Join(problems, "\n"))
}

// parameterCheck is what every parameter of one tool is held to.
type parameterCheck struct {
	tool       string
	registered map[string]struct{}
}

// parameterProblems checks every parameter at every depth: inside an object,
// and inside the objects a list holds. An id nested in a list of stops is
// sent by the same model as one at the top, and was left unchecked. A nested
// parameter may lean on the sentence of the list or object holding it for
// where its ids come from; a top-level one has only its own.
func parameterProblems(
	check parameterCheck,
	properties map[string]any,
	path, enclosing string,
) []string {
	var problems []string
	for param, raw := range properties {
		property, _ := raw.(map[string]any)
		at := param
		if path != "" {
			at = path + "." + param
		}
		text, _ := property["description"].(string)
		if strings.TrimSpace(text) == "" && path == "" {
			problems = append(problems, fmt.Sprintf("%s.%s: no description", check.tool, at))
			continue
		}
		problems = append(problems, recordMarkProblems(check.tool, at, property)...)
		if idParameter.MatchString(param) {
			if _, ok := selfEvident[param]; !ok && !namesSource(check, text) &&
				!namesSource(check, enclosing) {
				problems = append(problems, fmt.Sprintf(
					"%s.%s: does not name the tool (or the page or run subject) that supplies it",
					check.tool, at))
			}
		}
		for nestedPath, nested := range nestedProperties(property, at) {
			problems = append(problems, parameterProblems(check, nested, nestedPath, text)...)
		}
	}

	return problems
}

func namesSource(check parameterCheck, text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}

	return namesAnotherTool(text, check.tool, check.registered) || fromContext.MatchString(text)
}

// nestedProperties are the parameters an object holds, and those of the
// objects a list holds, keyed by the path a refusal would name them at.
func nestedProperties(property map[string]any, at string) map[string]map[string]any {
	out := make(map[string]map[string]any, 1)
	if nested, ok := property["properties"].(map[string]any); ok && len(nested) > 0 {
		out[at] = nested
	}
	if items, ok := property["items"].(map[string]any); ok {
		if nested, has := items["properties"].(map[string]any); has && len(nested) > 0 {
			out[at+"[]"] = nested
		}
	}

	return out
}

// recordMarkProblems holds a parameter marked with the resource its ids
// belong to, on itself or on its list's items, to a resource the runtime can
// check a prefix against. A mark naming anything else would refuse every id.
func recordMarkProblems(tool, at string, property map[string]any) []string {
	marks := make([]string, 0, 2)
	if resource := toolschema.RecordOf(property); resource != "" {
		marks = append(marks, resource)
	}
	if items, ok := property["items"].(map[string]any); ok {
		if resource := toolschema.RecordOf(items); resource != "" {
			marks = append(marks, resource)
		}
	}

	problems := make([]string, 0, len(marks))
	for _, resource := range marks {
		if _, known := permission.Resource(resource).IDPrefix(); !known {
			problems = append(problems, fmt.Sprintf(
				"%s.%s: marked as a %s id, which has no prefix in permission's table",
				tool, at, resource))
		}
	}

	return problems
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

// stagingSteps are writes that only stage work for a person to review before
// a later tool commits it, so a recipe may run one ahead of that tool.
var stagingSteps = map[string]struct{}{
	"build_invoice_run": {},
}

// A recipe and a prerequisite list name tools that exist. The order of a
// piece of work lives on the tools now, not in a template's instructions,
// and a misspelt step is told to every agent holding the tool and loads
// nothing.
//
// A recipe is followed as literally as it is written, so everything before
// the tool must only read: approve_billing_queue_items once listed
// assign_billing_queue_billers ahead of itself and post_invoices after, which
// told a model asked only to approve to reassign billers and post as well. A
// prerequisite is granted to every agent holding the tool, which is only safe
// for a read.
func TestEveryRecipeNamesRegisteredTools(t *testing.T) {
	t.Parallel()

	known := make(map[string]struct{})
	queries := make(map[string]struct{})
	for _, tool := range buildTools(t) {
		known[tool.Name()] = struct{}{}
		if _, ok := tool.(serviceports.AgentQueryTool); ok {
			queries[tool.Name()] = struct{}{}
		}
	}
	for _, policy := range agentruntime.RuntimePolicies() {
		known[policy.Name] = struct{}{}
	}

	var problems []string
	for _, tool := range buildTools(t) {
		steps := make([]string, 0, 8)
		if recipe, ok := tool.(serviceports.RecipeTool); ok {
			steps = append(steps, recipe.Recipe()...)
			problems = append(problems, recipeOrderProblems(
				tool.Name(), recipe.Recipe(), queries)...)
		}
		if needs, ok := tool.(serviceports.PrerequisiteTool); ok {
			steps = append(steps, needs.Prerequisites()...)
			for _, need := range needs.Prerequisites() {
				if _, ok := queries[need]; !ok {
					problems = append(problems, fmt.Sprintf(
						"%s: prerequisite %q is not a read", tool.Name(), need))
				}
			}
		}
		for _, step := range steps {
			if _, ok := known[step]; !ok {
				problems = append(problems, fmt.Sprintf(
					"%s: names %q, which is not a registered tool", tool.Name(), step))
			}
		}
	}

	sort.Strings(problems)
	require.Emptyf(t, problems, "%d recipes or prerequisites are wrong:\n%s",
		len(problems), strings.Join(problems, "\n"))
}

func recipeOrderProblems(
	name string,
	recipe []string,
	queries map[string]struct{},
) []string {
	if len(recipe) == 0 {
		return nil
	}
	at := slices.Index(recipe, name)
	if at < 0 {
		return []string{fmt.Sprintf("%s: its recipe does not name the tool itself", name)}
	}

	var problems []string
	for _, step := range recipe[:at] {
		if _, ok := queries[step]; ok {
			continue
		}
		if _, ok := stagingSteps[step]; ok {
			continue
		}
		problems = append(problems, fmt.Sprintf(
			"%s: its recipe runs %q, which writes, before the tool itself", name, step))
	}

	return problems
}

// leftOutMeansEvery is a description saying that leaving a parameter out
// widens the call to every record: "Leave it out to pay everything", "Leave
// out customerIds to assess every customer".
var (
	leftOutMeansEvery = regexp.MustCompile(
		`(?i)\b(leave (it|them) out|omit it|omit them|when left out)\b[^.]*\b(every|all|everything)\b`,
	)
	namedLeftOutMeansEvery = regexp.MustCompile(
		`(?i)\bleave out (\w+)(?: and (\w+))? to [^.]*\b(every|all|everything)\b`,
	)
)

/*
The runtime reads an optional parameter sent as "" or an empty list as not
sent, because several models fill every parameter a tool declares. For a
write whose parameter, left out, means every record, that turned "pay these
none" into "pay everything" and "assess these customers" into "assess every
customer". Such a parameter keeps its empty value (toolschema.KeepEmpty), and
a write that says leaving it out means every record must mark it.
*/
func TestEveryWriteWhoseOmissionWidensKeepsAnEmptyList(t *testing.T) {
	t.Parallel()

	var problems []string
	for _, tool := range buildTools(t) {
		if _, read := tool.(serviceports.AgentQueryTool); read {
			continue
		}
		properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
		widening := make(map[string]struct{}, 2)
		for _, match := range namedLeftOutMeansEvery.FindAllStringSubmatch(tool.Description(), -1) {
			for _, name := range match[1:3] {
				if name != "" {
					widening[name] = struct{}{}
				}
			}
		}
		for name, raw := range properties {
			property, _ := raw.(map[string]any)
			description, _ := property[toolschema.KeyDescription].(string)
			if leftOutMeansEvery.MatchString(description) {
				widening[name] = struct{}{}
			}
		}
		for name := range widening {
			property, declared := properties[name].(map[string]any)
			if !declared {
				continue
			}
			if keep, _ := property[toolschema.KeyKeepEmpty].(bool); !keep {
				problems = append(problems, fmt.Sprintf(
					"%s: %s widens to every record when left out but does not keep an "+
						"empty value (toolschema.KeepEmpty)", tool.Name(), name))
			}
		}
	}

	sort.Strings(problems)
	require.Emptyf(t, problems, "%d parameters would widen when sent empty:\n%s",
		len(problems), strings.Join(problems, "\n"))
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
