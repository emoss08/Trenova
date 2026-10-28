package fixtures

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deskRefs(t *testing.T) *deskSeedRefs {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return &deskSeedRefs{
		org: orgStub(orgID, buID),
		messages: []*inboundmessage.InboundMessage{
			{
				ID:             pulid.MustNew("imsg_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Status:         inboundmessage.StatusInReview,
				Subject:        "Load 88213 tender",
				FromAddress:    "dispatch@bigshipper.example",
				ReceivedAt:     1_800_000_000,
			},
		},
		proposals: []*agent.AgentProposal{
			{ID: pulid.MustNew("aprp_"), RunID: pulid.MustNew("arun_")},
			{ID: pulid.MustNew("aprp_"), RunID: pulid.MustNew("arun_")},
		},
		plans:      []*agent.AgentPlan{{ID: pulid.MustNew("apln_"), RunID: pulid.MustNew("arun_")}},
		agentNames: map[pulid.ID]string{},
		now:        1_800_000_000,
	}
}

func queueRefs(t *testing.T) *deskSeedRefs {
	t.Helper()

	refs := deskRefs(t)
	refs.readerID = pulid.MustNew("usr_")
	refs.shipments = map[string]*shipment.Shipment{}
	for _, pro := range []string{
		"SEED-SHP-001", "SEED-SHP-002", "SEED-SHP-004", "SEED-SHP-007", "SEED-SHP-008",
	} {
		refs.shipments[pro] = &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: pro}
	}
	refs.workers = []*worker.Worker{
		{ID: pulid.MustNew("wrk_"), FirstName: "Dana", LastName: "Ortiz"},
		{ID: pulid.MustNew("wrk_"), FirstName: "Tom", LastName: "Reyes"},
	}
	refs.definitions = map[agenttemplate.Template]*agentdefinition.Definition{}
	for _, template := range []agenttemplate.Template{
		agenttemplate.TemplateCarrierRiskDesk,
		agenttemplate.TemplateDetentionDesk,
		agenttemplate.TemplateCredentialDesk,
		agenttemplate.TemplateLoadMonitor,
	} {
		refs.definitions[template] = &agentdefinition.Definition{
			ID:       pulid.MustNew("adef_"),
			Template: template,
		}
	}

	return refs
}

func seededQueue(t *testing.T) (*deskSeedRefs, *queueSeed) {
	t.Helper()

	refs := queueRefs(t)
	return refs, (&DeskSeed{}).queueItems(refs, refs.readerID)
}

func lane(item *watchtower.Item, now int64) string {
	if item.DueAt == nil {
		return "fyi"
	}
	left := *item.DueAt - now
	switch {
	case left <= 4*3600:
		return "now"
	case left <= 24*3600:
		return "today"
	default:
		return "later"
	}
}

func TestDeskSeed_EveryQueueItemIsValid(t *testing.T) {
	_, queue := seededQueue(t)

	for _, item := range queue.items {
		multiErr := errortypes.NewMultiError()
		item.Validate(multiErr)
		assert.Falsef(t, multiErr.HasErrors(),
			"watchtower item %q is not valid: %v", item.Title, multiErr)
	}
}

func TestDeskSeed_QueueFillsEveryDeadlineLane(t *testing.T) {
	refs, queue := seededQueue(t)

	counts := map[string]int{}
	for _, item := range queue.items {
		counts[lane(item, refs.now)]++
	}

	assert.Equal(t, map[string]int{"now": 5, "today": 1, "later": 3, "fyi": 2}, counts)
}

func TestDeskSeed_QueueHasAgentsWorkingWaitingAndDone(t *testing.T) {
	_, queue := seededQueue(t)

	runs := make(map[pulid.ID]*agent.AgentRun, len(queue.runs))
	for _, run := range queue.runs {
		runs[run.ID] = run
	}
	statuses := map[watchtower.AgentStatus]int{}
	for _, item := range queue.items {
		if !item.HasAgent() {
			continue
		}
		run, ok := runs[item.AgentRunID]
		require.Truef(t, ok, "%q is handed to a run the seed does not write", item.Title)
		work := item.AgentWorkFrom(run)
		statuses[work.Status]++
		assert.Equal(t, watchtower.StatusInProgress, item.Status,
			"an item an agent works is in progress")
	}

	assert.Equal(t, map[watchtower.AgentStatus]int{
		watchtower.AgentStatusWorking:  2,
		watchtower.AgentStatusNeedsYou: 1,
		watchtower.AgentStatusDone:     1,
	}, statuses)
}

func TestDeskSeed_QueueCarriesAnOverdueDetentionClock(t *testing.T) {
	refs, queue := seededQueue(t)

	for _, item := range queue.items {
		if item.SourceKind != watchtower.SourceDetentionOccurrence {
			continue
		}
		require.NotNil(t, item.DueAt)
		assert.Less(t, *item.DueAt, refs.now, "the detention clock is already running")
		assert.Equal(t, "Detention is accruing", item.DueLabel)
		assert.Equal(t, watchtower.DueSourceSource, item.DueSource)
		return
	}
	t.Fatal("no detention item is seeded")
}

func TestDeskSeed_QueueHasTrailsAndItemsWithoutADeadline(t *testing.T) {
	_, queue := seededQueue(t)

	trails, undated := 0, 0
	for _, item := range queue.items {
		if len(item.Trail) > 0 {
			trails++
		}
		if item.DueAt == nil {
			undated++
			assert.Equal(t, watchtower.DueSourcePerson, item.DueSource,
				"an undated item says a person left it without one, so routing leaves it alone")
		}
	}

	assert.Equal(t, 4, trails)
	assert.Equal(t, 2, undated)
}

func TestDeskSeed_QueueMarksSomeItemsSeenForTheReader(t *testing.T) {
	refs, queue := seededQueue(t)

	seen := make(map[pulid.ID]bool, len(queue.reads))
	for _, read := range queue.reads {
		assert.Equal(t, refs.readerID, read.UserID)
		seen[read.ItemID] = true
	}
	unseen := 0
	for _, item := range queue.items {
		if !seen[item.ID] {
			unseen++
		}
	}

	assert.Equal(t, 4, unseen)
}

func TestDeskSeed_QueueSubjectsAreSeededRecords(t *testing.T) {
	refs, queue := seededQueue(t)

	known := map[pulid.ID]bool{}
	for _, entity := range refs.shipments {
		known[entity.ID] = true
	}
	for _, entity := range refs.workers {
		known[entity.ID] = true
	}
	for _, item := range queue.items {
		if item.SubjectID.IsNil() {
			assert.Emptyf(t, item.EventKind,
				"%q names an event to hand off on, but has no subject", item.Title)
			continue
		}
		assert.Truef(t, known[item.SubjectID], "%q is about a record the seed did not load",
			item.Title)
	}
}

func TestDeskSeed_TheStandingItemIsAlreadyResolved(t *testing.T) {
	items := (&DeskSeed{}).standingItems(deskRefs(t))

	require.Len(t, items, 1)
	assert.True(t, items[0].IsResolved())
	assert.Equal(t, watchtower.StatusResolved, items[0].Status)
	assert.Equal(t, watchtower.ResolutionCleared, items[0].ResolutionCode)
}

func TestDeskSeed_TheBriefingIsValid(t *testing.T) {
	entity := briefingFor(deskRefs(t))

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	require.Falsef(t, multiErr.HasErrors(), "briefing is not valid: %v", multiErr)

	// Un-narrated is the honest state: nothing wrote this prose, and a page
	// claiming to be narrated when no model wrote it is the one lie this
	// table must not tell.
	assert.False(t, entity.Narrated)
	assert.Equal(t, briefing.StatusReady, entity.Status)
}

// The headline and the sections are read together, so a figure that appears
// in one and disagrees in the other is the defect. The counts come from the
// refs rather than being written twice.
func TestDeskSeed_TheBriefingCountsAgreeWithWhatIsSeeded(t *testing.T) {
	refs := deskRefs(t)
	entity := briefingFor(refs)

	decisions := len(refs.proposals) + len(refs.plans)
	assert.Equal(t, decisions, entity.Facts["decisionsPending"])
	assert.Equal(t, len(refs.messages), entity.Facts["messagesInReview"])
	assert.Contains(t, entity.Headline, "3 decisions")

	for _, section := range entity.Sections {
		for _, item := range section.Items {
			assert.Truef(t, item.IsSafePath(),
				"section %q links to %q, which is not an application path",
				section.Title, item.Path)
		}
	}
}

func TestDeskSeed_EveryQueuePathIsAnApplicationPath(t *testing.T) {
	_, queue := seededQueue(t)

	for _, item := range queue.items {
		if item.Path == "" {
			continue
		}
		assert.Truef(t, stringutils.IsSafeAppPath(item.Path),
			"%q links to %q", item.Title, item.Path)
		assert.Truef(t, strings.HasPrefix(item.Path, "/"),
			"%q links to %q, which is not rooted", item.Title, item.Path)
	}
}
