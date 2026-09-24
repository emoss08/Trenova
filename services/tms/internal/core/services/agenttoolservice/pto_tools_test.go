package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePTODecider struct {
	approved  *repositories.UpdatePTOStatusRequest
	rejected  *repositories.UpdatePTOStatusRequest
	cancelled *repositories.UpdatePTOStatusRequest
}

func (f *fakePTODecider) Approve(
	_ context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	f.approved = req

	return &worker.WorkerPTO{ID: req.ID}, nil
}

func (f *fakePTODecider) Reject(
	_ context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	f.rejected = req

	return &worker.WorkerPTO{ID: req.ID}, nil
}

func (f *fakePTODecider) Cancel(
	_ context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	f.cancelled = req

	return &worker.WorkerPTO{ID: req.ID}, nil
}

func TestApproveWorkerPTO_CarriesTheRequestAndTheDecider(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	id := pulid.MustNew("wpto_").String()
	params := executeParams(map[string]any{"ptoId": id})

	require.NoError(t, newApproveWorkerPTOTool(pto).Execute(t.Context(), params))

	require.NotNil(t, pto.approved)
	assert.Equal(t, id, pto.approved.ID.String())
	assert.Equal(t, params.Actor.UserID, pto.approved.UserID)
	assert.Equal(t, params.OrganizationID, pto.approved.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, pto.approved.TenantInfo.BuID)
}

/*
The version is left for the service to read.

UpdatePTOStatusRequest carries an optimistic-concurrency version, and the
service fills it from the record when the caller sends zero. A model has no way
to know a version, so sending one would mean inventing it — and an invented
version either matches by luck or defeats the check that exists to catch two
people deciding the same request at once.
*/
func TestApproveWorkerPTO_LeavesTheExpectedVersionForTheService(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	require.NoError(t, newApproveWorkerPTOTool(pto).Execute(t.Context(), executeParams(
		map[string]any{"ptoId": pulid.MustNew("wpto_").String()},
	)))

	assert.Zero(t, pto.approved.ExpectedVersion)
}

/*
An agent principal cannot approve.

guardExecute refuses OpApprove to an agent, which is why this tool names the
operation honestly rather than calling a leave decision an update. A scheduled
agent running unattended must not decide a person's time off; a person
approving the proposal acts as themselves and passes.
*/
func TestApproveWorkerPTO_RefusesAnAgentPrincipal(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	params := executeParams(map[string]any{"ptoId": pulid.MustNew("wpto_").String()})
	params.Actor.PrincipalType = serviceports.PrincipalTypeAgent

	err := newApproveWorkerPTOTool(pto).Execute(t.Context(), params)

	require.ErrorIs(t, err, ErrAgentCannotApprove)
	assert.Nil(t, pto.approved)
}

// The worker is told why. A rejection composed by the model rather than
// reported to it would put words in a manager's mouth.
func TestRejectWorkerPTO_RequiresAReason(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	err := newRejectWorkerPTOTool(pto).Execute(t.Context(), executeParams(map[string]any{
		"ptoId": pulid.MustNew("wpto_").String(),
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reason")
	assert.Nil(t, pto.rejected, "nothing may be decided without the reason")
}

func TestRejectWorkerPTO_PassesTheReasonThrough(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	require.NoError(t, newRejectWorkerPTOTool(pto).Execute(t.Context(), executeParams(
		map[string]any{
			"ptoId":  pulid.MustNew("wpto_").String(),
			"reason": "Three drivers already off that week.",
		},
	)))

	assert.Equal(t, "Three drivers already off that week.", pto.rejected.Reason)
}

func TestCancelWorkerPTO_RequiresAReason(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	err := newCancelWorkerPTOTool(pto).Execute(t.Context(), executeParams(map[string]any{
		"ptoId": pulid.MustNew("wpto_").String(),
	}))

	require.Error(t, err)
	assert.Nil(t, pto.cancelled)
}

// A rejection is terminal and a cancellation returns the days but not the
// request. Only an approval can be undone, and the tiers have to say so.
func TestPTOTools_DeclareTheirAuthorizationAndReversibility(t *testing.T) {
	t.Parallel()

	approve := newApproveWorkerPTOTool(&fakePTODecider{})
	assert.Equal(t, permission.ResourceWorkerPTO, approve.Policy().Resource)
	assert.Equal(t, permission.OpApprove, approve.Policy().Operation)
	assert.True(t, approve.Policy().Reversible)

	reject := newRejectWorkerPTOTool(&fakePTODecider{})
	assert.Equal(t, permission.OpReject, reject.Policy().Operation)
	assert.False(t, reject.Policy().Reversible)

	cancel := newCancelWorkerPTOTool(&fakePTODecider{})
	assert.Equal(t, permission.OpCancel, cancel.Policy().Operation)
	assert.False(t, cancel.Policy().Reversible)
}

// None of the three may run without a person seeing it first.
func TestPTOTools_NeverAutoExecute(t *testing.T) {
	t.Parallel()

	for _, tool := range []serviceports.AgentTool{
		newApproveWorkerPTOTool(&fakePTODecider{}),
		newRejectWorkerPTOTool(&fakePTODecider{}),
		newCancelWorkerPTOTool(&fakePTODecider{}),
	} {
		assert.NotEqual(t, agent.TierAutoExecute, tool.Policy().DefaultTier, tool.Name())
	}
}

// A worker id sent where a request id belongs is the mistake to expect, so the
// descriptions say so and the parser refuses anything that is not an id.
func TestPTOTools_RefuseAnIdThatIsNotOne(t *testing.T) {
	t.Parallel()

	pto := &fakePTODecider{}
	err := newApproveWorkerPTOTool(pto).Execute(t.Context(), executeParams(map[string]any{
		"ptoId": "Maria Ortiz",
	}))

	require.Error(t, err)
	assert.Nil(t, pto.approved)
}
