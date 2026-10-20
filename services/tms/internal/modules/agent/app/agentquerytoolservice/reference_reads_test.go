package agentquerytoolservice_test

import (
	"errors"
	"strings"
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
func TestReferenceReads_RefuseExactlyWhatAuthorizeRefuses(t *testing.T) {
	t.Parallel()

	person := agentreadtest.Person()
	actors := map[string]*permissionports.RequestActor{
		"a person":            person,
		"an unattended agent": agentreadtest.Agent(person),
	}
	for _, name := range agentquerytoolservice.ReferenceReadNames() {
		resource := agentquerytoolservice.ReferenceReadResource(name)
		engine := &mocks.DenyResourcesPermissionEngine{Denied: []string{resource.String()}}
		reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine})
		tool, err := agentquerytoolservice.ReferenceReadTool(name, reader, engine)
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

// Every reference list names its typed parameters in its description, so a
// model choosing among tools sees what each narrows by.
func TestReferenceReads_DescribeEveryParameter(t *testing.T) {
	t.Parallel()

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine})
	for _, name := range agentquerytoolservice.ReferenceReadNames() {
		tool, err := agentquerytoolservice.ReferenceReadTool(name, reader, engine)
		require.NoError(t, err, name)

		properties, _ := tool.ParamSchema()["properties"].(map[string]any)
		for param := range properties {
			switch param {
			case "query", "limit", "after", "sortBy", "sortDirection":
				continue
			}
			named, ok := describedAs(param, properties)
			if !ok {
				continue
			}
			assert.Contains(t, tool.Description(), named, "%s does not name %s", name, param)
		}
	}
}

// describedAs is how a description names a parameter: a range is named
// once, by its lower end, as createdAtFrom/To or amountMin/Max.
func describedAs(param string, properties map[string]any) (string, bool) {
	ranges := [][2]string{{"From", "To"}, {"Min", "Max"}}
	for _, pair := range ranges {
		if base, ok := strings.CutSuffix(param, pair[0]); ok {
			return base + pair[0] + "/" + pair[1], true
		}
		if base, ok := strings.CutSuffix(param, pair[1]); ok {
			if _, paired := properties[base+pair[0]]; paired {
				return "", false
			}
		}
	}

	return param, true
}
