package conversation

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistantTurn_Validate_ChecksTraceID(t *testing.T) {
	t.Parallel()

	turn := &AssistantTurn{
		ThreadID: pulid.MustNew("ath_"),
		UserID:   pulid.MustNew("usr_"),
		Status:   AssistantTurnStatusRunning,
		Origin:   AssistantTurnOriginPerson,
	}

	multiErr := errortypes.NewMultiError()
	turn.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	turn.TraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	multiErr = errortypes.NewMultiError()
	turn.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	turn.TraceID = "4bf92f3577b34da6"
	multiErr = errortypes.NewMultiError()
	turn.Validate(multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "traceId", multiErr.Errors[0].Field)
}
