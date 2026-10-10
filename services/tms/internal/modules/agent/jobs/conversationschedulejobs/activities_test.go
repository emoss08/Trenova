package conversationschedulejobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/conversationschedulejobs/conversationschedulecontract"

	"github.com/emoss08/trenova/internal/modules/agent/app/conversationscheduleservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func fireActivity(
	t *testing.T,
	runner *fakeRunner,
	syncer *fakeSyncer,
	input *FireInput,
) (*conversationscheduleports.FireConversationScheduleResult, error) {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	activities := &Activities{runner: runner, syncer: syncer, l: zap.NewNop()}
	env.RegisterActivity(activities)
	value, err := env.ExecuteActivity(activities.FireConversationScheduleActivity, input)
	if err != nil {
		return nil, err
	}

	var result conversationscheduleports.FireConversationScheduleResult
	require.NoError(t, value.Get(&result))

	return &result, nil
}

/*
A slot fires the run into the service with its tenant and the instant it
fired, which is what keeps a retried attempt from asking twice.
*/
func TestFireActivity_HandsTheSlotToTheService(t *testing.T) {
	t.Parallel()

	turnID := pulid.MustNew("atrn_")
	runner := &fakeRunner{result: &conversationscheduleports.FireConversationScheduleResult{TurnID: turnID}}
	syncer := &fakeSyncer{}
	payload := &conversationschedulecontract.RunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		ScheduleID: pulid.MustNew("csch_"),
	}

	result, err := fireActivity(t, runner, syncer, &FireInput{Payload: payload, FiredAt: 1791203400})
	require.NoError(t, err)
	assert.Equal(t, turnID, result.TurnID)

	require.Len(t, runner.asked, 1)
	asked := runner.asked[0]
	assert.Equal(t, payload.ScheduleID, asked.ScheduleID)
	assert.Equal(t, payload.OrganizationID, asked.TenantInfo.OrgID)
	assert.Equal(t, payload.BusinessUnitID, asked.TenantInfo.BuID)
	assert.Equal(t, int64(1791203400), asked.FiredAt)
	assert.Empty(t, syncer.removed)
}

func TestFireActivity_RemovesTheScheduleOfADeletedRow(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{result: &conversationscheduleports.FireConversationScheduleResult{
		Skipped: conversationscheduleservice.SkipGone,
		Gone:    true,
	}}
	syncer := &fakeSyncer{}
	id := pulid.MustNew("csch_")

	_, err := fireActivity(t, runner, syncer, &FireInput{Payload: &conversationschedulecontract.RunPayload{ScheduleID: id}})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{id}, syncer.removed)
}

func TestFireActivity_ABusyConversationIsRetried(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{err: conversationscheduleservice.ErrConversationBusy}
	_, err := fireActivity(t, runner, &fakeSyncer{}, &FireInput{
		Payload: &conversationschedulecontract.RunPayload{ScheduleID: pulid.MustNew("csch_")},
	})
	var applicationErr *temporal.ApplicationError
	require.ErrorAs(t, err, &applicationErr)
	assert.Equal(t, "ConversationBusy", applicationErr.Type())
	assert.False(t, applicationErr.NonRetryable(), "the run waits for the reply in progress")
}

type fakeRunner struct {
	result *conversationscheduleports.FireConversationScheduleResult
	err    error
	asked  []conversationscheduleports.FireConversationScheduleRequest
}

func (f *fakeRunner) Fire(
	_ context.Context,
	req conversationscheduleports.FireConversationScheduleRequest,
) (*conversationscheduleports.FireConversationScheduleResult, error) {
	f.asked = append(f.asked, req)

	return f.result, f.err
}

type fakeSyncer struct {
	removed []pulid.ID
}

func (f *fakeSyncer) Sync(context.Context, *conversationschedule.Schedule) {}

func (f *fakeSyncer) Remove(_ context.Context, id pulid.ID) {
	f.removed = append(f.removed, id)
}
