package agenttoolcatalog_test

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/require"
)

var snakeWord = regexp.MustCompile(`\b[a-z]+(?:_[a-z0-9]+)+\b`)

// A tool's name is written out wherever it is named, so a rename leaves the
// old name in every description that pointed at it unless something looks.
// The document read was renamed document_get, and a description still sending
// the model to document_get_summary would send it to nothing. A word that reads
// as a tool name, a tool namespace then a verb some tool in it uses, must be a
// registered tool.
func TestEveryToolATextNamesIsRegistered(t *testing.T) {
	t.Parallel()

	tools := buildTools(t)
	known := make(map[string]struct{}, len(tools))
	verbs := make(map[string]map[string]struct{})
	for _, tool := range tools {
		known[tool.Name()] = struct{}{}
		namespace := agenttoolports.NamespaceOf(tool.Name())
		if namespace == "" {
			continue
		}
		rest := strings.TrimPrefix(tool.Name(), namespace+"_")
		verb, _, _ := strings.Cut(rest, "_")
		if verbs[namespace] == nil {
			verbs[namespace] = make(map[string]struct{})
		}
		verbs[namespace][verb] = struct{}{}
	}
	for _, policy := range runtimetools.Policies() {
		known[policy.Name] = struct{}{}
	}

	readsAsTool := func(word string) bool {
		namespace := agenttoolports.NamespaceOf(word)
		if namespace == "" {
			return false
		}
		verb, _, _ := strings.Cut(strings.TrimPrefix(word, namespace+"_"), "_")
		_, ok := verbs[namespace][verb]
		return ok
	}

	var problems []string
	for _, tool := range tools {
		texts := []string{tool.Description()}
		if searchable, ok := tool.(agenttoolports.SearchableTool); ok {
			texts = append(texts, searchable.SearchTerms()...)
		}
		texts = append(texts, schemaDescriptions(tool.ParamSchema())...)
		for _, text := range texts {
			for _, word := range snakeWord.FindAllString(text, -1) {
				if _, ok := known[word]; ok || !readsAsTool(word) {
					continue
				}
				problems = append(problems, fmt.Sprintf(
					"%s: names %q, which is not a registered tool", tool.Name(), word))
			}
		}
	}

	sort.Strings(problems)
	require.Emptyf(t, problems, "%d texts name a tool that does not exist:\n%s",
		len(problems), strings.Join(problems, "\n"))
}

// schemaDescriptions is every description a parameter schema carries, nested
// objects and array items included.
func schemaDescriptions(schema map[string]any) []string {
	var out []string
	var walk func(value any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, nested := range typed {
				if text, ok := nested.(string); ok && key == toolschema.KeyDescription {
					out = append(out, text)
					continue
				}
				walk(nested)
			}
		case []any:
			for _, nested := range typed {
				walk(nested)
			}
		}
	}
	walk(schema)

	return out
}
