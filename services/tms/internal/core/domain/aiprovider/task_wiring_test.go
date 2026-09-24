package aiprovider_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A task is routable only when every place that is keyed by it knows about it, and
nothing today notices when one is missed.

TaskQueryCompose is why this test exists. It shipped with natural-language
tables, was added to AllTasks(), and was left out of the provider-configuration
descriptors — so no administrator could tick it for a provider, ListForTask
returned nothing, and every Ask-a-table request failed with
ErrNoProviderConfigured. The feature was dead on arrival and the build was green.

These read the other files' source rather than importing them, because the
descriptors live in an API handler and the sampling table in an infrastructure
adapter; a domain test that imported either would invert the dependency it is
supposed to be checking.
*/

const (
	descriptorsFile = "../../../api/handlers/aiproviderhandler/catalog.go"
	samplingFile    = "../../../infrastructure/agentcompletion/modeladapter/sampling.go"
	schemaFile      = "../../../api/graphql/schema/aiprovider.graphqls"
)

// tasksNamedIn collects every aiprovider.TaskX selector mentioned in a Go file.
func tasksNamedIn(t *testing.T, path string) map[string]struct{} {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Clean(path), nil, parser.SkipObjectResolution)
	require.NoErrorf(t, err, "could not read %s", path)

	named := make(map[string]struct{})
	ast.Inspect(file, func(n ast.Node) bool {
		selector, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok || ident.Name != "aiprovider" {
			return true
		}
		if len(selector.Sel.Name) > 4 && selector.Sel.Name[:4] == "Task" {
			named[selector.Sel.Name] = struct{}{}
		}

		return true
	})

	return named
}

// goNameOf recovers the constant identifier for a task's wire value, so the
// failure message names the thing a person would go and add.
func goNameOf(task aiprovider.Task) string { return "Task" + string(task) }

// Without a descriptor a task cannot be assigned to a provider in the admin UI,
// so nothing serves it and every call fails at runtime.
func TestEveryTaskHasAProviderDescriptor(t *testing.T) {
	t.Parallel()

	declared := tasksNamedIn(t, descriptorsFile)
	for _, task := range aiprovider.AllTasks() {
		name := goNameOf(task)
		assert.Containsf(t, declared, name,
			"%s has no entry in taskDescriptors(); no provider can be assigned to it, "+
				"so every call routed to it fails with ErrNoProviderConfigured", name)
	}
}

// A task with no sampling case silently takes the 0.3 default. For anything
// that classifies or compiles, that means the same input can give two answers.
func TestEveryDeterministicTaskPinsItsTemperature(t *testing.T) {
	t.Parallel()

	// The tasks whose output is a label or a structure rather than prose. A
	// creative temperature on any of these is a bug somebody cannot reproduce.
	deterministic := []aiprovider.Task{
		aiprovider.TaskScopeClassification,
		aiprovider.TaskDocumentClassification,
		aiprovider.TaskDocumentExtraction,
		aiprovider.TaskQueryCompose,
		aiprovider.TaskInboundClassification,
		aiprovider.TaskEvaluationJudge,
	}

	named := tasksNamedIn(t, samplingFile)
	for _, task := range deterministic {
		name := goNameOf(task)
		assert.Containsf(t, named, name,
			"%s has no case in SamplingForTask, so it takes the 0.3 default; "+
				"a structured answer that changes between identical inputs is a bug", name)
	}
}

func TestEveryTaskIsNamedInSampling(t *testing.T) {
	t.Parallel()

	named := tasksNamedIn(t, samplingFile)
	for _, task := range aiprovider.AllTasks() {
		name := goNameOf(task)
		assert.Containsf(t, named, name,
			"%s is not named in SamplingForTask; give it a case, or an explicit "+
				"exemption when the task samples no tokens", name)
	}
}

func TestEveryTaskIsInTheGraphQLEnum(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile(filepath.Clean(schemaFile))
	require.NoErrorf(t, err, "could not read %s", schemaFile)

	block := regexp.MustCompile(`(?s)enum AITask \{(.*?)\}`).FindSubmatch(source)
	require.NotNil(t, block, "enum AITask not found in %s", schemaFile)

	declared := make(map[string]struct{})
	for _, line := range strings.Split(string(block[1]), "\n") {
		if value := strings.TrimSpace(line); value != "" {
			declared[value] = struct{}{}
		}
	}

	for _, task := range aiprovider.AllTasks() {
		assert.Containsf(t, declared, string(task),
			"%s is missing from the AITask GraphQL enum, so a provider serving it "+
				"cannot be read over GraphQL", task)
	}
}
