package aiproviderrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func renderDB() *bun.DB {
	return bun.NewDB(nil, pgdialect.New())
}

/*
ai_providers.tasks is text[], and bun renders a bare Go slice argument as JSON.
That made every routed call fail at the database:

	malformed array literal: "["AssistantChat"]" (SQLSTATE=22P02)

It is the whole router, not one feature: ListForTask is how any task finds a
provider, so an agent run, an assistant turn and an extraction all died the same
way. Rendering the query is the only way to catch it without a live PostgreSQL,
since a Go slice and a pgdialect.Array are the same type to the compiler.
*/
func TestListForTaskSendsAPostgresArrayNotJSON(t *testing.T) {
	t.Parallel()

	entities := make([]*aiprovider.Provider, 0, 1)
	sql := buildProvidersForTaskQuery(
		renderDB(),
		&entities,
		repositories.ListAIProvidersForTaskRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Task: aiprovider.TaskAssistantChat,
		},
	).String()

	assert.Contains(t, sql, `'{"AssistantChat"}'`, sql)
	assert.NotContains(t, sql, `["AssistantChat"]`,
		"a JSON array is what PostgreSQL rejects as a malformed array literal")
}

// The GIN index on tasks only serves containment, so the predicate has to stay
// @> rather than becoming = ANY(...), which would seq-scan.
func TestListForTaskUsesTheContainmentOperator(t *testing.T) {
	t.Parallel()

	entities := make([]*aiprovider.Provider, 0, 1)
	sql := buildProvidersForTaskQuery(
		renderDB(),
		&entities,
		repositories.ListAIProvidersForTaskRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Task: aiprovider.TaskAssistantChat,
		},
	).String()

	assert.Contains(t, sql, "@>", sql)
	assert.Contains(t, sql, "enabled", sql)
}
