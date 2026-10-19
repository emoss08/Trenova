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
		"a ceiling that is no tier": {
			Agent: string(agentdefinition.TemplateLoadMonitor), Schedule: true, Ceiling: "Sometimes",
		},
		"a tier that is no tier": {
			Agent: string(agentdefinition.TemplateLoadMonitor), Schedule: true,
			ToolTiers: map[string]agent.AutonomyTier{agentdefinition.CoreToolRaiseException: "Loud"},
		},
		"a tier on a tool the agent does not hold": {
			Agent: string(agentdefinition.TemplateLoadMonitor), Schedule: true,
			ToolTiers: map[string]agent.AutonomyTier{"invoice_post": agent.TierPropose},
		},
	}
	for name, fire := range cases {
		assert.Error(t, fireScenario(fire).Validate(), name)
	}

	shaped := &Fire{
		Agent:   string(agentdefinition.TemplateBillingException),
		Event:   agent.EventBillingQueueItemException,
		Subject: "select id from billing_queue_items limit 1",
		Ceiling: agent.TierPropose,
		ToolTiers: map[string]agent.AutonomyTier{
			agentdefinition.CoreToolRaiseException: agent.TierPropose,
			agentdefinition.CoreToolFlagForReview:  agent.TierPropose,
		},
		StopWhileWaiting: true,
	}
	require.NoError(t, fireScenario(shaped).Validate(), "a core tool's tier may be set")
	assert.Equal(t, agent.TierPropose, shaped.ceiling(agentdefinition.TemplateBillingException))
	assert.Equal(t, agentdefinition.TemplateBillingException.StarterCeiling(),
		event.ceiling(agentdefinition.TemplateBillingException))

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

func TestUnattendedMatches_ResetsToolTiersALeftRunSet(t *testing.T) {
	t.Parallel()

	template := agentdefinition.TemplateBillingException
	existing := unattendedShape(template)
	assert.True(t, unattendedMatches(existing, unattendedShape(template)))

	existing.ToolTiers = map[string]agent.AutonomyTier{
		agentdefinition.CoreToolRaiseException: agent.TierPropose,
	}
	assert.False(t, unattendedMatches(existing, unattendedShape(template)),
		"tiers a crashed case left on the agent are cleared by the next sync")
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
			{ID: "c1", Name: "billing_get_queue_item", Arguments: map[string]any{"id": "bqi_1"}},
			{ID: "c2", Name: "shipment_place_hold"},
		}},
		{Role: "tool", ToolCallID: "c1", ToolName: "billing_get_queue_item", ToolVerdict: "ran"},
		{Role: "tool", ToolCallID: "c2", ToolName: "shipment_place_hold", ToolVerdict: "proposed"},
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

func TestCaseErrors_NamesTheStepAFireFailedIn(t *testing.T) {
	t.Parallel()

	result := &CaseResult{
		Error: "setup 2: duplicate key value violates unique constraint",
		Steps: []*StepResult{
			{Index: 1, Kind: "fire", Error: "the subject query returned 0 rows, not one"},
			{Index: 2, Kind: "say"},
			{Index: 3, Kind: "decide", Error: "no pending proposal to approve"},
		},
	}

	assert.Equal(t, []string{
		"setup 2: duplicate key value violates unique constraint",
		"step 1: the subject query returned 0 rows, not one",
		"step 3: no pending proposal to approve",
	}, result.Errors())
	assert.Empty(t, (&CaseResult{Steps: []*StepResult{{Index: 1}}}).Errors())
}

type runRecordsFake struct {
	agentports.AgentRunRepository

	run        *agent.AgentRun
	transcript *agent.RunTranscript
}

func (f *runRecordsFake) GetByID(
	_ context.Context,
	req agentports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	if req.ID != f.run.ID {
		return nil, assert.AnError
	}
	run := *f.run

	return &run, nil
}

func (f *runRecordsFake) ListTranscriptsByIDs(
	_ context.Context,
	req agentports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	runs := make([]*agent.AgentRun, 0, len(req.IDs))
	for _, id := range req.IDs {
		if id == f.run.ID {
			runs = append(runs, &agent.AgentRun{ID: id, Transcript: f.transcript})
		}
	}

	return runs, nil
}

func TestAwaitLoop_EndsOnTheStatusTheRunRecords(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("arun_")
	records := &runRecordsFake{
		run: &agent.AgentRun{
			ID:      runID,
			Status:  agent.RunStatusAwaitingDecision,
			Summary: "Filed an exception for a person.",
		},
		transcript: &agent.RunTranscript{
			Messages: []agent.TranscriptMessage{{Role: "Assistant", Content: "Done."}},
		},
	}
	bench := &Bench{deps: deps{RunRecords: records}}
	session := &Session{
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	fired := &firedRun{run: &agent.AgentRun{ID: runID}}

	ctx, cancel := context.WithTimeout(t.Context(), runPollInterval/2)
	defer cancel()

	run, err := bench.awaitLoop(ctx, session, fired)
	require.NoError(t, err, "a run already awaiting a decision ends the wait on the first read")
	assert.Equal(t, agent.RunStatusAwaitingDecision, run.Status)
	assert.Equal(t, "Filed an exception for a person.", run.Summary)

	transcript, err := bench.loadTranscript(t.Context(), session, runID)
	require.NoError(t, err)
	assert.Equal(t, records.transcript, transcript)
}
