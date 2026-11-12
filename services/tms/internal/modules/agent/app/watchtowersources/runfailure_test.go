package watchtowersources

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A run records the error of whatever broke. The person relying on the agent
is told what happened in their own words, never the workflow's or the
provider's, and never with a record id in it.
*/
func TestRunFailureReason_SaysWhatHappenedInAPersonsWords(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"The provider returned no completion after four attempts: context deadline exceeded.": "The AI provider did not answer in time.",
		"activity error (type: ModelCallActivity, scheduledEventID: 30, startedEventID: 31, " +
			"identity: trenova-tms-local): every configured chat provider failed: provider " +
			"request failed (status 400)": "The AI provider could not take the request.",
		"provider request failed (status 429): rate limit":      "The AI provider is over its limit and turned the request away.",
		"activity error (type: RecordActivity): nil pointer":    "The run stopped on an error. Open the run to see what it recorded.",
		"The shipment ar_01M717A3AHMP2CBW5SSW8KZ5K1 was closed": "The shipment was closed",
		"": "",
	}
	for recorded, want := range cases {
		assert.Equal(t, want, RunFailureReason(recorded), recorded)
	}
}

// A failed run's item names the agent and why, and carries no run id: its
// link opens the run.
func TestDescribeFailedRun_NamesNoRunID(t *testing.T) {
	t.Parallel()

	run := &agent.AgentRun{
		ID:           pulid.MustNew("ar_"),
		ErrorMessage: "context deadline exceeded",
		StartedAt:    100,
	}

	item := DescribeFailedRun(run, "Billing exceptions")

	assert.Equal(t, "The AI provider did not answer in time.", item.Summary)
	for _, fact := range item.Facts {
		assert.NotContains(t, fact.Value, run.ID.String())
		assert.NotEqual(t, "Run", fact.Label)
	}
	assert.Contains(t, item.Path, run.ID.String(), "the link still opens the run")
}

// A run that recorded no error says so. Its summary is what it did, not why
// it stopped, so it is not shown as the reason.
func TestDescribeFailedRun_ARunWithNoErrorDoesNotOfferItsSummaryAsTheReason(t *testing.T) {
	t.Parallel()

	item := DescribeFailedRun(&agent.AgentRun{
		ID:      pulid.MustNew("ar_"),
		Summary: "Reviewed three held invoices.",
	}, "Billing exceptions")

	assert.Equal(
		t,
		"The run stopped without recording why. Open the run to see how far it got.",
		item.Summary,
	)
	assert.Equal(t, []watchtower.Fact{{Label: "Agent", Value: "Billing exceptions"}},
		watchtower.NormalizeFacts(item.Facts), "a reason that is not known is not a fact")
}

// An agent that fails several runs in a row is one problem on the feed.
func TestLatestFailurePerAgent_KeepsEachAgentsNewestFailure(t *testing.T) {
	t.Parallel()

	billing, dispatch := pulid.MustNew("agdef_"), pulid.MustNew("agdef_")
	runs := []*agent.AgentRun{
		{ID: "ar_1", AgentDefinitionID: billing, CreatedAt: 10},
		{ID: "ar_2", AgentDefinitionID: dispatch, CreatedAt: 20},
		nil,
		{ID: "ar_3", AgentDefinitionID: billing, CreatedAt: 30},
	}

	latest := latestFailurePerAgent(runs)

	require.Len(t, latest, 2)
	assert.Equal(t, pulid.ID("ar_3"), latest[0].ID)
	assert.Equal(t, pulid.ID("ar_2"), latest[1].ID)
}

func TestExceptionTitle_SaysTheCategoryInWords(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "An agent needs a person: service failure",
		exceptionTitle(agentexception.ExceptionCategory("ServiceFailure")))
	assert.Equal(t, "An agent needs a person", exceptionTitle(agentexception.CategoryOther))
}

// An inbound message's title says what it is in words, and one classed as
// nothing in particular is just a message.
func TestInboundMessageTitle_SaysTheClassificationInWords(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Rate confirmation needs review: Rate con",
		inboundMessageTitle(&inboundmessage.InboundMessage{
			Classification: inboundmessage.ClassificationRateConfirmation,
			Subject:        "Rate con",
		}))
	assert.Equal(t, "Message needs review: (no subject)",
		inboundMessageTitle(&inboundmessage.InboundMessage{
			Classification: inboundmessage.ClassificationOther,
		}))
	assert.Equal(t, "Message needs review: Hello",
		inboundMessageTitle(&inboundmessage.InboundMessage{Subject: "Hello"}))
}
