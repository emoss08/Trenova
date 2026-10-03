package conversation

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validThread() *Thread {
	return &Thread{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		UserID:            pulid.MustNew("usr_"),
		AgentDefinitionID: pulid.MustNew("agd_"),
		Status:            ThreadStatusActive,
		Origin:            ThreadOriginDesk,
	}
}

// A conversation opened from a record carries the record as a pair: a type
// without an id, or an id without a type, names nothing.
func TestThreadValidate_SubjectIsAPair(t *testing.T) {
	t.Parallel()

	halfTyped := validThread()
	halfTyped.SubjectType = agent.SubjectShipment
	multiErr := errortypes.NewMultiError()
	halfTyped.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	halfID := validThread()
	halfID.SubjectID = pulid.MustNew("shp_")
	multiErr = errortypes.NewMultiError()
	halfID.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	whole := validThread()
	whole.SubjectType = agent.SubjectShipment
	whole.SubjectID = pulid.MustNew("shp_")
	multiErr = errortypes.NewMultiError()
	whole.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, whole.HasSubject())
	assert.False(t, validThread().HasSubject())
}

func TestThreadValidate_OriginMustBeKnown(t *testing.T) {
	t.Parallel()

	thread := validThread()
	thread.Origin = "Sidebar"
	multiErr := errortypes.NewMultiError()
	thread.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

// A quick question from the palette is not listed until it is kept; every
// other origin is a conversation from the start.
func TestThreadOrigin_Listed(t *testing.T) {
	t.Parallel()

	assert.False(t, ThreadOriginAsk.Listed())
	for _, origin := range []ThreadOrigin{ThreadOriginPanel, ThreadOriginDesk, ThreadOriginWatchtower, ThreadOriginBriefing} {
		assert.True(t, origin.Listed(), string(origin))
		assert.True(t, origin.IsValid())
	}
	assert.False(t, ThreadOrigin("Nope").IsValid())
}

func TestThreadOrigin_PageBoundConversationsAreNeitherListedNorKept(t *testing.T) {
	t.Parallel()

	for _, origin := range []ThreadOrigin{ThreadOriginImport, ThreadOriginFormula} {
		assert.True(t, origin.IsValid(), string(origin))
		assert.True(t, origin.PageBound(), string(origin))
		assert.False(t, origin.Listed(), string(origin))
		assert.False(t, origin.Keepable(), string(origin))
	}
	assert.True(t, ThreadOriginAsk.Keepable())
	assert.False(t, ThreadOriginAsk.PageBound())
	for _, origin := range []ThreadOrigin{
		ThreadOriginPanel, ThreadOriginDesk, ThreadOriginWatchtower, ThreadOriginBriefing,
	} {
		assert.False(t, origin.Keepable(), string(origin))
		assert.False(t, origin.PageBound(), string(origin))
	}
	assert.ElementsMatch(t,
		[]ThreadOrigin{ThreadOriginAsk, ThreadOriginImport, ThreadOriginFormula},
		UnlistedOrigins(),
	)
	for _, origin := range AllThreadOrigins() {
		assert.True(t, origin.IsValid(), string(origin))
	}
}

// Pinned facts are kept as the person meant them: tidied, without blanks or
// repeats, in the order they were pinned.
func TestNormalizePinnedFacts_TidiesWithoutReordering(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]string{"Invoice date is Oct 3", "Acme pays net 45"},
		NormalizePinnedFacts([]string{
			"  Invoice   date is Oct 3 ", "", "Acme pays net 45", "Invoice date is Oct 3",
		}),
	)
	assert.Empty(t, NormalizePinnedFacts(nil))
}

// The facts ride in every turn's system prompt, so the list and each fact
// are capped.
func TestThreadValidate_CapsPinnedFacts(t *testing.T) {
	t.Parallel()

	tooMany := validThread()
	for i := range MaxPinnedFacts + 1 {
		tooMany.PinnedFacts = append(tooMany.PinnedFacts, strings.Repeat("x", i+1))
	}
	multiErr := errortypes.NewMultiError()
	tooMany.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	tooLong := validThread()
	tooLong.PinnedFacts = []string{strings.Repeat("x", MaxPinnedFactLength+1)}
	multiErr = errortypes.NewMultiError()
	tooLong.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	fine := validThread()
	fine.PinnedFacts = []string{"Invoice date is Oct 3"}
	multiErr = errortypes.NewMultiError()
	fine.Validate(multiErr)
	require.False(t, multiErr.HasErrors())
}
