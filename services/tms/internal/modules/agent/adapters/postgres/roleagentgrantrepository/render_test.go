package roleagentgrantrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestGrantedAgentIDsQueryReadsEachAgentOnce(t *testing.T) {
	t.Parallel()

	ctx := keeltest.TenantContext(t)
	sql, args := keeltest.RenderIn(
		t,
		ctx,
		grantedAgentIDsQuery(ctx, permissionports.ListGrantedAgentIDsRequest{
			OrganizationID: keeltest.TestOrganizationID,
			RoleIDs:        []pulid.ID{"rol_1"},
		}),
	)
	assert.Contains(t, sql, `GROUP BY`)
	assert.Contains(t, sql, `"agent_definition_id"`)
	assert.Contains(t, args, keeltest.TestOrganizationID.String())
}

func TestCurrentGrantsQueryLocksTheGrants(t *testing.T) {
	t.Parallel()

	rag := rbacdb.RoleAgentGrants
	sql, _ := keeltest.Render(t, currentGrantsQuery(&replaceParams{
		fixed:   rag.RoleID,
		fixedID: pulid.ID("rol_1"),
		varying: rag.AgentDefinitionID,
	}))
	assert.Contains(t, sql, "FOR UPDATE")
}

func TestRolesQueryReadsEveryRoleWithoutIDs(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, rolesQuery(nil))
	assert.NotContains(t, sql, "ANY(")
	assert.Contains(t, sql, `"name" ASC`)
}
