package aiproviderrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

/*
ai_providers.tasks is text[]: a task has to reach PostgreSQL as an array, not
as JSON, or every routed call fails at the database with a malformed array
literal. ListForTask is how any task finds a provider, so an agent run, an
assistant turn and an extraction all depend on it.
*/
func TestListForTaskSendsTheTaskAsATextArray(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, providersForTaskQuery(aitask.TaskAssistantChat))

	assert.Contains(t, sql, `."tasks" @> $`, sql)
	assert.Contains(t, args, []string{string(aitask.TaskAssistantChat)})
}

// The GIN index on tasks only serves containment, so the predicate has to stay
// @> rather than becoming = ANY(...), which would seq-scan.
func TestListForTaskUsesTheContainmentOperator(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, providersForTaskQuery(aitask.TaskAssistantChat))

	assert.Contains(t, sql, "@>", sql)
	assert.NotContains(t, sql, "ANY(", sql)
	assert.Contains(t, sql, `."enabled" IS TRUE`, sql)
}

// A picker of providers offers only those that are on and serve the task, so a
// person cannot prefer a provider the router would never try for it.
func TestSelectOptionsFilterKeepsEnabledProvidersThatServeTheTask(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, selectOptionsQuery(aitask.TaskAssistantChat))

	assert.Contains(t, sql, `."enabled" IS TRUE`, sql)
	assert.Contains(t, sql, `."tasks" @> $`, sql)
	assert.Contains(t, sql, `."priority" ASC`, sql)
}

func TestSelectOptionsFilterWithoutATaskKeepsEveryEnabledProvider(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, selectOptionsQuery(""))

	assert.Contains(t, sql, `."enabled" IS TRUE`, sql)
	assert.NotContains(t, sql, "@>", sql)
}
