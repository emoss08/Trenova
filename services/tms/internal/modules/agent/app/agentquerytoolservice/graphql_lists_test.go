package agentquerytoolservice_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A reference list refused its resource is refused through GraphQL, for a
// person and for an unattended agent, exactly as authorize refuses it, and
// a refusal is told apart from a failure.
func TestConvertedLists_RefuseExactlyWhatAuthorizeRefuses(t *testing.T) {
	t.Parallel()

	person := agentreadtest.Person()
	actors := map[string]*permissionports.RequestActor{
		"a person":            person,
		"an unattended agent": agentreadtest.Agent(person),
	}
	for _, name := range agentquerytoolservice.ConvertedListNames() {
		resource := agentquerytoolservice.ConvertedListResource(name)
		engine := &mocks.DenyResourcesPermissionEngine{Denied: []string{resource.String()}}
		reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine})
		tool, err := agentquerytoolservice.ConvertedListTool(name, reader, engine)
		require.NoError(t, err, name)

		for actorName, actor := range actors {
			t.Run(name+"/"+actorName, func(t *testing.T) {
				t.Parallel()

				_, readErr := tool.Query(readscope.With(t.Context(), nil),
					&agentports.QueryToolParams{
						OrganizationID: actor.OrganizationID,
						BusinessUnitID: actor.BusinessUnitID,
						Actor:          actor,
						Timezone:       "America/Chicago",
						Params:         map[string]any{"limit": 5},
						RunID:          pulid.MustNew("ar_"),
					})

				var refused *agentports.ReadRefusedError
				require.True(t, errors.As(readErr, &refused), "got %v", readErr)
				assert.Equal(t, resource, refused.Resource)
			})
		}
	}
}

// Every converted list names, in its description, every typed parameter an
// agent at the default data access can use, so a model choosing among tools
// sees what each narrows by.
func TestConvertedLists_DescribeEveryParameter(t *testing.T) {
	t.Parallel()

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine})
	for _, name := range agentquerytoolservice.ConvertedListNames() {
		tool, err := agentquerytoolservice.ConvertedListTool(name, reader, engine)
		require.NoError(t, err, name)

		properties, _ := tool.ParamSchema()["properties"].(map[string]any)
		for param := range properties {
			mention, required := agentquerytoolservice.ExpectedMention(tool, param)
			if !required {
				continue
			}
			assert.Contains(t, tool.Description(), mention, "%s does not name %s", name, param)
		}
	}
}
