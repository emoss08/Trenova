package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/productguide"
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
		users:         &stubUsers{user: &tenant.User{}},
		runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{},
			nil,
		),
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

type stubPageGuide struct {
	serviceports.ProductGuide

	pages map[string]*productguide.Page
}

func (s *stubPageGuide) PageForPath(path string) (*productguide.Page, bool) {
	page, ok := s.pages[path]

	return page, ok
}

type stubUsers struct {
	repositories.UserRepository

	user *tenant.User
}

func (s *stubUsers) GetByID(
	context.Context,
	repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	return s.user, nil
}

func guideBuilder(guide serviceports.ProductGuide) *ContextBuilder {
	return &ContextBuilder{
		logger:        zap.NewNop(),
		organizations: &stubOrganizations{org: &tenant.Organization{Name: "Acme"}},
		users:         &stubUsers{user: &tenant.User{Name: "Dana Dispatcher"}},
		runtime:       newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil),
		guide:         guide,
	}
}

// A path says nothing to a model about what the person is looking at; the
// guide's name, place and purpose for it do.
func TestContextBuilder_DescribesThePageFromTheGuide(t *testing.T) {
	t.Parallel()

	builder := guideBuilder(&stubPageGuide{pages: map[string]*productguide.Page{
		"/billing/invoices?item=inv_1": {
			Path:        "/billing/invoices",
			Name:        "Invoices",
			Breadcrumb:  []string{"Billing", "Invoices"},
			Description: "Every invoice.",
		},
	}})

	rc, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Trigger:    agent.RunTriggerChat,
		Page:       &agentdefinition.PageContext{Path: "/billing/invoices?item=inv_1"},
	})
	require.NoError(t, err)

	require.NotNil(t, rc.PageGuide)
	assert.Equal(t, "Invoices", rc.PageGuide.Name)
	assert.Equal(t, "Billing › Invoices", rc.PageGuide.Location)
	assert.Equal(t, "Every invoice.", rc.PageGuide.Summary, "the header line stands in for a summary")
	assert.True(t, rc.Guide)
}

// The guide tools are withheld from a run nobody is watching, so its prompt
// must not tell it to use them.
func TestContextBuilder_OffersTheGuideOnlyToAPersonInAConversation(t *testing.T) {
	t.Parallel()

	builder := guideBuilder(&stubPageGuide{})

	scheduled, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Trigger:    agent.RunTriggerScheduled,
	})
	require.NoError(t, err)
	assert.False(t, scheduled.Guide)
	assert.Nil(t, scheduled.PageGuide)

	actor := testActor()
	actor.PrincipalType = serviceports.PrincipalTypeAgent
	byAgent, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: testDefinition(),
		Actor:      actor,
		Trigger:    agent.RunTriggerChat,
	})
	require.NoError(t, err)
	assert.False(t, byAgent.Guide)
}
