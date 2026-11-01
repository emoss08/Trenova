package deskscheduleresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func text(value string) *string {
	return &value
}

func TestScheduleEditReadsAnEditOnlyWhole(t *testing.T) {
	t.Parallel()

	enabled := true
	edit, err := scheduleEdit(&gqlmodel.UpdateAssistantScheduleInput{Enabled: &enabled})
	require.NoError(t, err)
	assert.Nil(t, edit, "pausing or resuming edits nothing")

	edit, err = scheduleEdit(&gqlmodel.UpdateAssistantScheduleInput{
		Prompt: text("Send me delivered shipments"),
		Repeat: text("friday"),
		Time:   text("08:00"),
	})
	require.NoError(t, err)
	assert.Equal(t, &conversationschedule.Edit{
		Prompt: "Send me delivered shipments",
		Repeat: "friday",
		Time:   "08:00",
	}, edit)

	_, err = scheduleEdit(&gqlmodel.UpdateAssistantScheduleInput{Prompt: text("Only the words")})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Len(t, multiErr.Errors, 2, "an edit names what to ask, how often and when together")

	_, err = scheduleEdit(&gqlmodel.UpdateAssistantScheduleInput{})
	assert.True(t, errortypes.IsError(err), "a change that changes nothing is refused")
}

func TestScheduleOperationsCheckTheRESTGrants(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("csch_").String()
	run := resolvertest.RefusingEngine()
	_, err := (&MutationResolver{&Deps{Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: run}}}).
		RunAssistantSchedule(
			resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
			id,
		)
	resolvertest.RequireRefused(t, err, run, permission.ResourceAssistant, permission.OpCreate)

	remove := resolvertest.RefusingEngine()
	_, err = (&MutationResolver{&Deps{Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: remove}}}).
		DeleteAssistantSchedule(
			resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
			id,
		)
	resolvertest.RequireRefused(t, err, remove, permission.ResourceAssistant, permission.OpRead)
}

func TestSchedulesAreRefusedWhenTheServiceIsNotAvailable(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{Core: &resolvercore.Core{
		L: zap.NewNop(),
		PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
			permission.ResourceAssistant.String() + "|" + string(permission.OpRead): true,
		}},
	}}}

	_, err := resolver.AssistantSchedules(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		nil,
		nil,
	)

	assert.True(t, errortypes.IsBusinessError(err))
}
