package deskbench

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fireScenario(fire *Fire) *Scenario {
	return &Scenario{Name: "unattended", Steps: []Step{{Fire: fire}}}
}

func TestFire_Validation(t *testing.T) {
	t.Parallel()

	event := &Fire{
		Agent:   string(agentdefinition.TemplateBillingException),
		Event:   agent.EventBillingQueueItemException,
		Subject: "select id from billing_queue_items limit 1",
	}
	require.NoError(t, fireScenario(event).Validate())
	assert.True(t, fireScenario(event).Unattended())

	schedule := &Fire{Agent: string(agentdefinition.TemplateLoadMonitor), Schedule: true}
	require.NoError(t, fireScenario(schedule).Validate())

	cases := map[string]*Fire{
		"an event the template does not answer": {
			Agent: string(agentdefinition.TemplateBillingException), Event: agent.EventShipmentCreated,
			Subject: "select 1",
		},
		"an event run with no subject": {
			Agent: string(agentdefinition.TemplateBillingException),
			Event: agent.EventBillingQueueItemException,
		},
		"a scheduled agent fired by an event": {
			Agent: string(agentdefinition.TemplateLoadMonitor), Event: agent.EventShipmentCreated,
		},
		"a chat agent":        {Agent: string(agentdefinition.TemplateDispatchAssistant), Schedule: true},
		"an unknown template": {Agent: "NoSuchDesk", Schedule: true},
		"a subject that writes": {
			Agent: string(agentdefinition.TemplateBillingException),
			Event: agent.EventBillingQueueItemException, Subject: "delete from shipments",
		},
	}
	for name, fire := range cases {
		assert.Error(t, fireScenario(fire).Validate(), name)
	}

	twoSteps := fireScenario(schedule)
	twoSteps.Steps = append(twoSteps.Steps, Step{Say: "and then?"})
	require.Error(t, twoSteps.Validate())

	graded := &Scenario{Name: "chat", Steps: []Step{{Say: "hi", Expect: Expect{Run: "Completed"}}}}
	require.Error(t, graded.Validate(), "run belongs on a fire step")
}

func TestUnattendedTemplates_CoverEveryEventAndScheduledTemplate(t *testing.T) {
	t.Parallel()

	templates := UnattendedTemplates()
	assert.Contains(t, templates, agentdefinition.TemplateLoadMonitor)
	assert.Contains(t, templates, agentdefinition.TemplateDispatchAssignment)
	assert.Contains(t, templates, agentdefinition.TemplateEDIDesk)
	for _, template := range templates {
		trigger := template.StarterTrigger()
		assert.True(t,
			trigger == agentdefinition.TriggerEvent || trigger == agentdefinition.TriggerScheduled,
			template)
		shape := unattendedShape(template)
		assert.True(t, shape.LearningOff)
		assert.False(t, shape.Enabled, "kept off between cases, so nothing real starts it")
		assert.Equal(t, agentdefinition.OutputReport, shape.OutputMode)
		if trigger == agentdefinition.TriggerScheduled {
			assert.Equal(t, unattendedCron, shape.CronExpression)
		}
	}
}

func TestRecorderPin_HoldsARunToItsCell(t *testing.T) {
	t.Parallel()

	recorder := NewRecorder(nil)
	definitionID, providerID := pulid.MustNew("agd_"), pulid.MustNew("aiprv_")
	release := recorder.Pin(definitionID, ProviderPin{
		ProviderID: providerID,
		Effort:     aiprovider.ReasoningLow,
	})

	req := &agentports.ChatCompletionRequest{
		Attribution: aiusageports.AIUsageAttribution{AgentDefinitionID: definitionID},
	}
	recorder.pin(req)
	assert.Equal(t, providerID, req.PreferredProviderID)
	assert.True(t, req.PinPreferred)
	assert.Equal(t, aiprovider.ReasoningLow, req.Reasoning)

	release()
	other := &agentports.ChatCompletionRequest{
		Attribution: aiusageports.AIUsageAttribution{AgentDefinitionID: definitionID},
	}
	recorder.pin(other)
	assert.False(t, other.PinPreferred)
}

func TestTranscriptTools_PairsCallsWithTheirResults(t *testing.T) {
	t.Parallel()

	transcript := &agent.RunTranscript{Messages: []agent.TranscriptMessage{
		{Role: "assistant", ToolCalls: []agent.TranscriptToolCall{
			{ID: "c1", Name: "get_billing_queue_item", Arguments: map[string]any{"id": "bqi_1"}},
			{ID: "c2", Name: "place_shipment_hold"},
		}},
		{Role: "tool", ToolCallID: "c1", ToolName: "get_billing_queue_item", ToolVerdict: "ran"},
		{Role: "tool", ToolCallID: "c2", ToolName: "place_shipment_hold", ToolVerdict: "proposed"},
	}}

	tools := transcriptTools(transcript)
	require.Len(t, tools, 2)
	assert.Equal(t, "ran", tools[0].Verdict)
	assert.True(t, tools[0].Finished)
	assert.True(t, tools[1].Proposed)
}

func TestRunOutcome_SubstitutesItsIDs(t *testing.T) {
	t.Parallel()

	outcome := &runOutcome{ID: pulid.MustNew("ar_"), Subject: pulid.MustNew("bqi_")}
	got := outcome.substitute("select 1 from x where run_id = {{run}} and id = {{subject}}")
	assert.Contains(t, got, "'"+outcome.ID.String()+"'")
	assert.Contains(t, got, "'"+outcome.Subject.String()+"'")

	var none *runOutcome
	assert.Equal(t, "select 1", none.substitute("select 1"))
}

type fakeProviders struct {
	agentports.AIProviderRepository
	provider *aiprovider.Provider
}

func (f *fakeProviders) GetByID(
	context.Context,
	agentports.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	return f.provider, nil
}

type fakeReference struct{}

func (fakeReference) Prices(
	_ context.Context,
	_ aiprovider.Kind,
	ids []string,
) map[string]agentports.ReferencePrice {
	output := decimal.RequireFromString("8")
	read := decimal.RequireFromString("0.1")

	return map[string]agentports.ReferencePrice{ids[0]: {
		InputCostPerMillion:     decimal.RequireFromString("1"),
		OutputCostPerMillion:    &output,
		CacheReadCostPerMillion: &read,
	}}
}

func TestPricer_PricesAnUnpricedProviderFromTheListThenTheReference(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	haiku := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Kind: aiprovider.KindAnthropicMessages, Model: "claude-haiku-5-5",
	}
	listed := newPricer(&fakeProviders{provider: haiku}, fakeReference{})
	price := listed.price(t.Context(), tenant, haiku.ID, "claude-haiku-5-5",
		aiprovider.TokenUsage{Input: 1_000_000, CacheRead: 1_000_000, Output: 1_000_000}, nil)
	require.NotNil(t, price.cost)
	assert.Equal(t, PriceSourceListPrice, price.source)
	assert.True(t, price.cost.Equal(decimal.RequireFromString("3.05")), price.cost.String(),
		"above 100K prompt tokens Haiku 5.5 is priced five times over")

	luna := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Kind: aiprovider.KindOpenAIResponses, Model: "gpt-6-luna",
	}
	referenced := newPricer(&fakeProviders{provider: luna}, fakeReference{})
	price = referenced.price(t.Context(), tenant, luna.ID, "gpt-6-luna",
		aiprovider.TokenUsage{Input: 1_000_000, CacheRead: 500_000, Output: 100_000}, nil)
	require.NotNil(t, price.cost)
	assert.Equal(t, PriceSourceReference, price.source)
	assert.True(t, price.cost.Equal(decimal.RequireFromString("1.35")), price.cost.String())

	charged := decimal.RequireFromString("0.5")
	price = referenced.price(t.Context(), tenant, luna.ID, "gpt-6-luna", aiprovider.TokenUsage{},
		&charged)
	assert.Equal(t, PriceSourceProvider, price.source)
}

func TestPricer_PricesEachModelAProviderServedOnItsOwn(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	gateway := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Kind: aiprovider.KindAnthropicMessages, Model: "claude-haiku-5-5",
	}
	pricer := newPricer(&fakeProviders{provider: gateway}, fakeReference{})
	usage := aiprovider.TokenUsage{Input: 1_000, Output: 1_000}

	haiku := pricer.price(t.Context(), tenant, gateway.ID, "claude-haiku-5-5", usage, nil)
	opus := pricer.price(t.Context(), tenant, gateway.ID, "claude-opus-5-5", usage, nil)
	require.NotNil(t, haiku.cost)
	require.NotNil(t, opus.cost)
	assert.True(t, opus.cost.GreaterThan(*haiku.cost),
		"a failover to another model on the same provider is priced at that model's rate")
}
