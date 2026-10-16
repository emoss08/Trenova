package assistanthandler

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
PATCH /assistant/schedules/:id/ pauses or resumes a schedule with enabled, and
edits it with prompt, repeat and time, which come together: an edit missing
one of them is refused by name rather than half applied.
*/
func TestUpdateScheduleRequest_ReadsAnEditOnlyWhole(t *testing.T) {
	t.Parallel()

	paused := updateScheduleRequest{Enabled: new(bool)}
	edit, err := paused.edit()
	require.NoError(t, err)
	assert.Nil(t, edit, "a pause is no edit")

	whole := updateScheduleRequest{
		Prompt: stringutils.Ptr("Send me delivered shipments"),
		Repeat: stringutils.Ptr("friday"),
		Time:   stringutils.Ptr("10:15"),
	}
	edit, err = whole.edit()
	require.NoError(t, err)
	assert.Equal(t, &conversationschedule.Edit{
		Prompt: "Send me delivered shipments",
		Repeat: "friday",
		Time:   "10:15",
	}, edit)

	partial := updateScheduleRequest{Prompt: stringutils.Ptr("Send me delivered shipments")}
	_, err = partial.edit()
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, field := range multiErr.Errors {
		fields = append(fields, field.Field)
	}
	assert.Equal(t, []string{"repeat", "time"}, fields)
}
