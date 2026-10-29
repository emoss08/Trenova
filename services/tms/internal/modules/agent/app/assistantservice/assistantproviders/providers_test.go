package assistantproviders

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProviderRepo struct {
	aiproviderports.AIProviderRepository

	captured aiproviderports.ListAIProvidersForTaskRequest
	items    []*aiprovider.Provider
	err      error
}

func (f *fakeProviderRepo) ListForTask(
	_ context.Context,
	req aiproviderports.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	f.captured = req

	return f.items, f.err
}

/*
The picker must not become a way to read credentials.

A provider row carries an encrypted API key and the endpoint it belongs to. The
administrative list is gated on ResourceAIProvider, which a dispatcher does not
hold; this one is gated on the assistant's own read permission, so it has to
project rather than return the entity.
*/
func TestSelectableProviders_CarriesNoCredentials(t *testing.T) {
	t.Parallel()

	repo := &fakeProviderRepo{items: []*aiprovider.Provider{{
		ID:      pulid.MustNew("aiprv_"),
		Name:    "OpenRouter free",
		Kind:    aiprovider.KindOpenAIChat,
		Model:   "inclusionai/ling-3.0-flash-vl:free",
		APIKey:  "sk-do-not-leak-this",
		BaseURL: "https://openrouter.ai/api/v1",
		Enabled: true,
		Tasks:   []aitask.Task{aitask.TaskAssistantChat},
	}}}

	options, err := (&Options{Providers: repo}).
		SelectableProviders(t.Context(), assistanttest.ProviderActor())
	require.NoError(t, err)
	require.Len(t, options, 1)

	assert.Equal(t, "inclusionai/ling-3.0-flash-vl:free", options[0].Model)
	assert.Equal(t, "OpenRouter free", options[0].Name)
}

// The candidate list is the router's own, so the offer is scoped to the
// caller's tenant and to the assistant task rather than to every provider the
// organization has configured.
func TestSelectableProviders_AsksOnlyForThisTenantsChatProviders(t *testing.T) {
	t.Parallel()

	repo := &fakeProviderRepo{}
	actor := assistanttest.ProviderActor()

	_, err := (&Options{Providers: repo}).SelectableProviders(t.Context(), actor)
	require.NoError(t, err)

	assert.Equal(t, aitask.TaskAssistantChat, repo.captured.Task)
	assert.Equal(t, actor.OrganizationID, repo.captured.TenantInfo.OrgID)
	assert.Equal(t, actor.BusinessUnitID, repo.captured.TenantInfo.BuID)
}

// A disabled provider is not a choice, even if a stale list still names it.
func TestSelectableProviders_SkipsWhatCannotServeTheTask(t *testing.T) {
	t.Parallel()

	repo := &fakeProviderRepo{items: []*aiprovider.Provider{
		{
			ID: pulid.MustNew("aiprv_"), Name: "Disabled", Enabled: false,
			Tasks: []aitask.Task{aitask.TaskAssistantChat},
		},
		{
			ID: pulid.MustNew("aiprv_"), Name: "Usable", Enabled: true,
			Tasks: []aitask.Task{aitask.TaskAssistantChat},
		},
	}}

	options, err := (&Options{Providers: repo}).
		SelectableProviders(t.Context(), assistanttest.ProviderActor())
	require.NoError(t, err)

	require.Len(t, options, 1)
	assert.Equal(t, "Usable", options[0].Name)
}

/*
A choice is kept only while it is still on offer.

Providers get deleted, disabled, and unassigned from the assistant after
somebody picked one. The router already ignores an id it cannot resolve and
answers from the priority order — but leaving the dead id on the thread would
keep showing a model in the picker that nothing will ever use again.
*/
func TestResolvePreference_DropsAChoiceThatIsNoLongerOffered(t *testing.T) {
	t.Parallel()

	offered := pulid.MustNew("aiprv_")
	repo := &fakeProviderRepo{items: []*aiprovider.Provider{{
		ID: offered, Enabled: true, Tasks: []aitask.Task{aitask.TaskAssistantChat},
	}}}
	service := &Options{Providers: repo}

	assert.Equal(t, offered,
		service.ResolvePreference(t.Context(), offered, assistanttest.ProviderActor()))
	assert.True(
		t,
		service.ResolvePreference(t.Context(), pulid.MustNew("aiprv_"), assistanttest.ProviderActor()).
			IsNil(),
		"an id that is not on offer must not survive on the thread",
	)
}

// No choice is the default and must stay empty rather than becoming a lookup.
func TestResolvePreference_LeavesAnEmptyChoiceAlone(t *testing.T) {
	t.Parallel()

	repo := &fakeProviderRepo{}
	resolved := (&Options{Providers: repo}).
		ResolvePreference(t.Context(), pulid.Nil, assistanttest.ProviderActor())

	assert.True(t, resolved.IsNil())
	assert.Empty(t, repo.captured.Task, "an absent choice needs no provider lookup")
}
