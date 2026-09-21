package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubOrganizations struct {
	repositories.OrganizationRepository

	org   *tenant.Organization
	calls int
}

func (s *stubOrganizations) GetByID(
	context.Context,
	repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	s.calls++

	return s.org, nil
}

// The organization's timezone is what every tool means by "today". It used to
// be read only when the prompt wanted the organization's name, so an agent
// whose providers left that out ran every date filter on a UTC clock.
func TestContextBuilder_ResolvesTheTimezoneWhateverThePromptWants(t *testing.T) {
	t.Parallel()

	orgs := &stubOrganizations{org: &tenant.Organization{Name: "Acme", Timezone: "America/Denver"}}
	builder := &ContextBuilder{
		logger:        zap.NewNop(),
		organizations: orgs,
		runtime:       newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil),
	}

	// An empty provider list means every provider. This is the set that used
	// to leave the tools on a UTC clock: nothing here asks for the organization.
	definition := testDefinition()
	definition.ContextProviders = []agentdefinition.ContextProvider{agentdefinition.ContextPage}

	rc, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      testActor(),
	})
	require.NoError(t, err)

	assert.Equal(t, "America/Denver", rc.Timezone)
	assert.Empty(t, rc.OrganizationName, "the name still follows the prompt's providers")
	assert.Equal(t, 1, orgs.calls)
}
