package agenttoolcatalog_test

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxToolsOutsideANamespace = 0

var (
	namespaceName  = regexp.MustCompile(`^[a-z]+$`)
	namespacedTool = regexp.MustCompile(`^[a-z]+(_[a-z0-9]+)+$`)
	dateParameter  = regexp.MustCompile(
		`(At|Date|On|Day)$|^(windowStart|windowEnd|from|to|since|until)$`,
	)
)

var renamedFreeText = map[string]string{
	"notes":        "note",
	"cancelReason": "reason",
	"search":       "query",
	"q":            "query",
	"term":         "query",
}

func TestEveryNamespaceIsOneWordWithASummary(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	for _, namespace := range agenttoolports.ToolNamespaces() {
		assert.Regexp(t, namespaceName, namespace.Name)
		assert.NotEmptyf(t, namespace.Title, "%s has no title", namespace.Name)
		assert.NotEmptyf(t, namespace.Summary, "%s has no summary for the namespace map",
			namespace.Name)
		_, dup := seen[namespace.Name]
		assert.Falsef(t, dup, "%s is listed twice", namespace.Name)
		seen[namespace.Name] = struct{}{}
	}
}

func TestToolsMoveIntoNamespacesAndNeverOut(t *testing.T) {
	t.Parallel()

	outside := make([]string, 0, 512)
	for _, tool := range buildTools(t) {
		if agenttoolports.NamespaceOf(tool.Name()) == "" {
			outside = append(outside, tool.Name())
		}
	}
	sort.Strings(outside)

	require.LessOrEqualf(t, len(outside), maxToolsOutsideANamespace,
		"a new tool is named <namespace>_<verb>[_<object>] with a namespace from "+
			"agenttoolports.ToolNamespaces; %d tools are outside one:\n%s",
		len(outside), strings.Join(outside, "\n"))
	require.Equalf(t, maxToolsOutsideANamespace, len(outside),
		"lower maxToolsOutsideANamespace to %d: tools moved into a namespace", len(outside))
}

func TestNamespacedToolsFollowTheConventions(t *testing.T) {
	t.Parallel()

	var problems []string
	for _, tool := range buildTools(t) {
		name := tool.Name()
		if agenttoolports.NamespaceOf(name) == "" {
			continue
		}
		if !namespacedTool.MatchString(name) {
			problems = append(problems, name+": not <namespace>_<verb>[_<object>] in lower case")
		}
		properties, _ := tool.ParamSchema()["properties"].(map[string]any)
		problems = append(problems, conventionProblems(name, properties, "")...)
	}

	slices.Sort(problems)
	require.Emptyf(t, problems, "%d parameters break the tool conventions "+
		"(docs/engineering/agent-tool-conventions.md):\n%s",
		len(problems), strings.Join(problems, "\n"))
}

func conventionProblems(tool string, properties map[string]any, path string) []string {
	var problems []string
	for param, raw := range properties {
		property, _ := raw.(map[string]any)
		at := param
		if path != "" {
			at = path + "." + param
		}
		if want, renamed := renamedFreeText[param]; renamed {
			problems = append(problems, fmt.Sprintf("%s.%s: name it %s", tool, at, want))
		}
		kind := toolschema.DeclaredType(property)
		if param == "limit" && kind == toolschema.TypeInteger {
			if _, bounded := property[toolschema.KeyMaximum]; !bounded {
				problems = append(problems, fmt.Sprintf("%s.%s: declare its maximum", tool, at))
			}
		}
		if param == "after" {
			if _, paged := properties["limit"]; !paged {
				problems = append(problems, fmt.Sprintf("%s.%s: an after pages a limit", tool, at))
			}
			if _, offset := properties["offset"]; offset {
				problems = append(problems, fmt.Sprintf(
					"%s.%s: a list pages by after or by offset, never both", tool, at))
			}
		}
		if param == "offset" {
			if _, paged := properties["limit"]; !paged {
				problems = append(problems, fmt.Sprintf("%s.%s: an offset pages a limit", tool, at))
			}
		}
		if kind == toolschema.TypeString && dateParameter.MatchString(param) &&
			len(toolschema.EnumValues(property)) == 0 {
			if format, _ := property[toolschema.KeyFormat].(string); format == "" {
				problems = append(problems, fmt.Sprintf(
					"%s.%s: a date takes agenttoolschema.Date, DateTime or LocalDateTime",
					tool, at))
			}
		}
		if nested, ok := property["properties"].(map[string]any); ok {
			problems = append(problems, conventionProblems(tool, nested, at)...)
		}
		if items, ok := property["items"].(map[string]any); ok {
			if nested, has := items["properties"].(map[string]any); has {
				problems = append(problems, conventionProblems(tool, nested, at+"[]")...)
			}
		}
	}

	return problems
}
