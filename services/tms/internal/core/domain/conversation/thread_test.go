package conversation

import (
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
