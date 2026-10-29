package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimecontext"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/productguideports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The organization's timezone is what every tool means by "today". It used to
// be read only when the prompt wanted the organization's name, so an agent
// whose providers left that out ran every date filter on a UTC clock.
func TestContextBuilder_ResolvesTheTimezoneWhateverThePromptWants(t *testing.T) {
	t.Parallel()

	orgs := &runtimetest.StubOrganizations{
		Org: &tenant.Organization{Name: "Acme", Timezone: "America/Denver"},
	}
	builder := runtimecontext.NewBuilder(runtimecontext.BuilderParams{
		Logger:        zap.NewNop(),
		Organizations: orgs,
		Users:         &runtimetest.StubUsers{User: &tenant.User{}},
		Runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{},
			nil,
		),
	})

	// An empty provider list means every provider. This is the set that used
	// to leave the tools on a UTC clock: nothing here asks for the organization.
	definition := runtimetest.Definition()
	definition.ContextProviders = []agentdefinition.ContextProvider{agentdefinition.ContextPage}

	rc, err := builder.Build(t.Context(), &agentruntimeports.RuntimeContextRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
	})
	require.NoError(t, err)

	assert.Equal(t, "America/Denver", rc.Timezone)
	assert.Empty(t, rc.OrganizationName, "the name still follows the prompt's providers")
	assert.Equal(t, 1, orgs.Calls)
}

type stubPageGuide struct {
	productguideports.ProductGuide

	pages map[string]*productguide.Page
}

func (s *stubPageGuide) PageForPath(path string) (*productguide.Page, bool) {
	page, ok := s.pages[path]

	return page, ok
}

func guideBuilder(guide productguideports.ProductGuide) agentruntimeports.RuntimeContextBuilder {
	return runtimecontext.NewBuilder(runtimecontext.BuilderParams{
		Logger:        zap.NewNop(),
		Organizations: &runtimetest.StubOrganizations{Org: &tenant.Organization{Name: "Acme"}},
		Users:         &runtimetest.StubUsers{User: &tenant.User{Name: "Dana Dispatcher"}},
		Runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{},
			nil,
		),
		Guide: guide,
	})
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

	rc, err := builder.Build(t.Context(), &agentruntimeports.RuntimeContextRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Trigger:    agent.RunTriggerChat,
		Page:       &agentturn.PageContext{Path: "/billing/invoices?item=inv_1"},
	})
	require.NoError(t, err)

	require.NotNil(t, rc.PageGuide)
	assert.Equal(t, "Invoices", rc.PageGuide.Name)
	assert.Equal(t, "Billing › Invoices", rc.PageGuide.Location)
	assert.Equal(
		t,
		"Every invoice.",
		rc.PageGuide.Summary,
		"the header line stands in for a summary",
	)
	assert.True(t, rc.Guide)
}

// The guide tools are withheld from a run nobody is watching, so its prompt
// must not tell it to use them.
func TestContextBuilder_OffersTheGuideOnlyToAPersonInAConversation(t *testing.T) {
	t.Parallel()

	builder := guideBuilder(&stubPageGuide{})

	scheduled, err := builder.Build(t.Context(), &agentruntimeports.RuntimeContextRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Trigger:    agent.RunTriggerScheduled,
	})
	require.NoError(t, err)
	assert.False(t, scheduled.Guide)
	assert.Nil(t, scheduled.PageGuide)

	actor := runtimetest.Actor()
	actor.PrincipalType = permissionports.PrincipalTypeAgent
	byAgent, err := builder.Build(t.Context(), &agentruntimeports.RuntimeContextRequest{
		Definition: runtimetest.Definition(),
		Actor:      actor,
		Trigger:    agent.RunTriggerChat,
	})
	require.NoError(t, err)
	assert.False(t, byAgent.Guide)
}
