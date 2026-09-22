package development

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
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

// The tower stores a path and opens it; a path that is not an application
// path is refused, and a seeded row carrying one would fail at db-seed.
func TestDeskSeed_EveryStandingItemIsValid(t *testing.T) {
	seed := NewDeskSeed()

	for _, item := range seed.standingItems(deskRefs(t)) {
		multiErr := errortypes.NewMultiError()
		item.Validate(multiErr)
		assert.Falsef(t, multiErr.HasErrors(),
			"watchtower item %q is not valid: %v", item.Title, multiErr)
	}
}

// A standing item stands in for a record that is not seeded. Offering to hand
// one to an agent would be offering to work on nothing, so none may carry a
// subject — and that is a property of the data, not of the UI that reads it.
func TestDeskSeed_StandingItemsNameNoSubject(t *testing.T) {
	for _, item := range NewDeskSeed().standingItems(deskRefs(t)) {
		assert.Emptyf(t, item.SubjectType,
			"%q names a subject, but nothing is seeded behind it", item.Title)
		assert.Truef(t, item.SubjectID.IsNil(),
			"%q names a subject id, but nothing is seeded behind it", item.Title)
	}
}

// The feed is only worth looking at if it spans severities: one colour
// everywhere shows nothing about how the feed reads.
func TestDeskSeed_StandingItemsSpanSeverities(t *testing.T) {
	seen := map[watchtower.Severity]bool{}
	for _, item := range NewDeskSeed().standingItems(deskRefs(t)) {
		seen[item.Severity] = true
	}

	for _, severity := range []watchtower.Severity{
		watchtower.SeverityInfo,
		watchtower.SeverityWarning,
		watchtower.SeverityCritical,
	} {
		assert.Truef(t, seen[severity], "no standing item is %s", severity)
	}
}

// One resolved item, so the "unresolved only" filter has something to hide.
func TestDeskSeed_OneStandingItemIsAlreadyResolved(t *testing.T) {
	resolved := 0
	for _, item := range NewDeskSeed().standingItems(deskRefs(t)) {
		if item.IsResolved() {
			resolved++
		}
	}

	assert.Equal(t, 1, resolved,
		"the feed needs exactly one resolved row for the filter to be worth trying")
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

// Every path the seed writes is checked the same way the store checks it.
func TestDeskSeed_EveryStandingPathIsAnApplicationPath(t *testing.T) {
	for _, item := range NewDeskSeed().standingItems(deskRefs(t)) {
		if item.Path == "" {
			continue
		}
		assert.Truef(t, stringutils.IsSafeAppPath(item.Path),
			"%q links to %q", item.Title, item.Path)
		assert.Truef(t, strings.HasPrefix(item.Path, "/"),
			"%q links to %q, which is not rooted", item.Title, item.Path)
	}
}
