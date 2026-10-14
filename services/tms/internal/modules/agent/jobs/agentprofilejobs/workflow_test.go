package agentprofilejobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentprofilejobs/agentprofilecontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	return env
}

func orgRef() agentprofileports.SubjectRef {
	return agentprofileports.OrganizationSubject(
		pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")},
	)
}

func readyPlan(req *agentprofileports.GenerateRequest) *agentprofileports.GenerationPlan {
	return &agentprofileports.GenerationPlan{
		Ref:     req.Ref,
		Trigger: req.Trigger,
		Digest:  "digest",
		Request: &agentports.StructuredCompletionRequest{TenantInfo: req.Ref.Tenant()},
	}
}

func written() *agentprofileports.GenerationOutcome {
	return &agentprofileports.GenerationOutcome{
		ProfileID: pulid.ID("aprf_a"),
		Status:    agentprofile.StatusCurrent,
	}
}

func TestRequestWorkflow_APersonAskingWritesAtOnce(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	var preparedAt time.Time

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *agentprofileports.GenerateRequest) (*agentprofileports.GenerationPlan, error) {
			preparedAt = env.Now()
			return readyPlan(req), nil
		}).Once()
	env.OnActivity(a.ProfileModelActivity, mock.Anything, mock.Anything).
		Return(&ModelReply{Result: &agentports.StructuredCompletionResult{Text: "{}"}}, nil).Once()
	env.OnActivity(a.FinishProfileActivity, mock.Anything, mock.Anything).
		Return(written(), nil).Once()
	env.OnActivity(a.PruneProfileActivity, mock.Anything, mock.Anything).Return(nil).Once()

	started := env.Now()
	env.ExecuteWorkflow(RequestWorkflow, &agentprofilecontract.RequestInput{
		Request: agentprofileports.GenerateRequest{
			Ref:     orgRef(),
			Trigger: agentprofile.TriggerRequested,
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.Less(t, preparedAt.Sub(started), time.Second)

	var result ProfileResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Written)
	env.AssertExpectations(t)
}

func TestRequestWorkflow_GuidanceWaitsForQuietAndCostsOneWrite(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	var preparedAt time.Time

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(agentprofilecontract.RequestedSignalName, agentprofileports.GenerateRequest{
			Ref:     orgRef(),
			Trigger: agentprofile.TriggerGuidance,
		})
	}, time.Minute)

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *agentprofileports.GenerateRequest) (*agentprofileports.GenerationPlan, error) {
			preparedAt = env.Now()
			return readyPlan(req), nil
		}).Once()
	env.OnActivity(a.ProfileModelActivity, mock.Anything, mock.Anything).
		Return(&ModelReply{Result: &agentports.StructuredCompletionResult{Text: "{}"}}, nil).Once()
	env.OnActivity(a.FinishProfileActivity, mock.Anything, mock.Anything).
		Return(written(), nil).Once()
	env.OnActivity(a.PruneProfileActivity, mock.Anything, mock.Anything).Return(nil).Once()

	started := env.Now()
	env.ExecuteWorkflow(RequestWorkflow, &agentprofilecontract.RequestInput{
		Request: agentprofileports.GenerateRequest{
			Ref:     orgRef(),
			Trigger: agentprofile.TriggerGuidance,
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.GreaterOrEqual(t, preparedAt.Sub(started),
		time.Minute+agentprofilecontract.GuidanceQuietPeriod,
		"a line given while it waited starts the quiet period again")
	env.AssertExpectations(t)
}

func TestProfileWorkflow_NoProviderWritesNothingAndFailsNothing(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *agentprofileports.GenerateRequest) (*agentprofileports.GenerationPlan, error) {
			return readyPlan(req), nil
		}).Once()
	env.OnActivity(a.ProfileModelActivity, mock.Anything, mock.Anything).
		Return(&ModelReply{NoProvider: true}, nil).Once()

	env.ExecuteWorkflow(ProfileWorkflow, &agentprofileports.GenerateRequest{Ref: orgRef()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ProfileResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, ProfileResult{Skipped: 1}, result)
	env.AssertExpectations(t)
}

func TestProfileWorkflow_AModelThatKeepsFailingIsRecorded(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	var finished *agentprofileports.FinishGenerationRequest

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *agentprofileports.GenerateRequest) (*agentprofileports.GenerationPlan, error) {
			return readyPlan(req), nil
		}).Once()
	env.OnActivity(a.ProfileModelActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("refused", "ModelRefused", errors.New("refused")))
	env.OnActivity(a.FinishProfileActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *agentprofileports.FinishGenerationRequest) (*agentprofileports.GenerationOutcome, error) {
			finished = req
			return &agentprofileports.GenerationOutcome{
				ProfileID: pulid.ID("aprf_f"),
				Status:    agentprofile.StatusFailed,
			}, nil
		}).Once()
	env.OnActivity(a.PruneProfileActivity, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(ProfileWorkflow, &agentprofileports.GenerateRequest{Ref: orgRef()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.NotNil(t, finished)
	assert.Contains(t, finished.Error, "refused")
	assert.Nil(t, finished.Result)

	var result ProfileResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Failed)
}

func TestProfileWorkflow_ASkippedPlanCallsNoModel(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(&agentprofileports.GenerationPlan{Ref: orgRef(), Skip: agentprofileports.SkipUnchanged}, nil).
		Once()

	env.ExecuteWorkflow(ProfileWorkflow, &agentprofileports.GenerateRequest{Ref: orgRef()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestMerge_APersonAskingOutranksGuidance(t *testing.T) {
	t.Parallel()

	person := pulid.ID("usr_a")
	merged := agentprofileports.GenerateRequest{Ref: orgRef(), Trigger: agentprofile.TriggerGuidance}
	merge(&merged, &agentprofileports.GenerateRequest{
		Ref: orgRef(), Trigger: agentprofile.TriggerRequested, RequestedByID: person,
	})
	assert.Equal(t, agentprofile.TriggerRequested, merged.Trigger)
	assert.Equal(t, person, merged.RequestedByID)

	merge(&merged, &agentprofileports.GenerateRequest{
		Ref: orgRef(), Trigger: agentprofile.TriggerGuidance,
	})
	assert.Equal(t, agentprofile.TriggerRequested, merged.Trigger)
}

func TestRequestWorkflow_TheRequestThatStartedItIsHeardOnce(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	request := agentprofileports.GenerateRequest{
		Ref:           orgRef(),
		Trigger:       agentprofile.TriggerRequested,
		RequestedByID: pulid.ID("usr_a"),
	}

	// SignalWithStartWorkflow hands the same request over twice: as the start
	// argument and as the first signal.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(agentprofilecontract.RequestedSignalName, request)
	}, 0)

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *agentprofileports.GenerateRequest) (*agentprofileports.GenerationPlan, error) {
			return readyPlan(req), nil
		}).Once()
	env.OnActivity(a.ProfileModelActivity, mock.Anything, mock.Anything).
		Return(&ModelReply{Result: &agentports.StructuredCompletionResult{Text: "{}"}}, nil).Once()
	env.OnActivity(a.FinishProfileActivity, mock.Anything, mock.Anything).
		Return(written(), nil).Once()
	env.OnActivity(a.PruneProfileActivity, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(RequestWorkflow, &agentprofilecontract.RequestInput{Request: request})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ProfileResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Written, "one click is one write")
	env.AssertExpectations(t)
}

func TestOrganizationWorkflow_TheOrganizationsFailureIsNotItsPeoples(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	env.RegisterWorkflow(ProfileWorkflow)
	var a *Activities
	person := pulid.ID("usr_a")

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *agentprofileports.GenerateRequest) (*agentprofileports.GenerationPlan, error) {
			if req.Ref.Subject == agentprofile.SubjectOrganization {
				return nil, temporal.NewNonRetryableApplicationError("facts", "Broken", errors.New("facts"))
			}
			return readyPlan(req), nil
		})
	env.OnActivity(a.ListActiveUsersActivity, mock.Anything, mock.Anything).
		Return([]pulid.ID{person}, nil).Once()
	env.OnActivity(a.ProfileModelActivity, mock.Anything, mock.Anything).
		Return(&ModelReply{Result: &agentports.StructuredCompletionResult{Text: "{}"}}, nil).Once()
	env.OnActivity(a.FinishProfileActivity, mock.Anything, mock.Anything).
		Return(written(), nil).Once()
	env.OnActivity(a.PruneProfileActivity, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(OrganizationWorkflow, &OrganizationInput{
		TenantWorkItem: temporaljobs.NewTenantWorkItem(orgRef().Tenant(), 1),
		Now:            1_790_000_000,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result OrganizationResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, 1, result.Failed, "the organization's profile")
	assert.Equal(t, 1, result.Written, "the person's profile was still written")
	assert.Equal(t, 1, result.People)
	env.AssertExpectations(t)
}

func TestOrganizationWorkflow_AnOrganizationThatWritesNoProfilesAsksNobody(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	env.RegisterWorkflow(ProfileWorkflow)
	var a *Activities

	env.OnActivity(a.PrepareProfileActivity, mock.Anything, mock.Anything).
		Return(&agentprofileports.GenerationPlan{
			Ref:  orgRef(),
			Skip: agentprofileports.SkipLearningOff,
		}, nil).Once()

	env.ExecuteWorkflow(OrganizationWorkflow, &OrganizationInput{
		TenantWorkItem: temporaljobs.NewTenantWorkItem(orgRef().Tenant(), 1),
		Now:            1_790_000_000,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result OrganizationResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Zero(t, result.People, "its people were not listed, let alone written about")
	env.AssertExpectations(t)
}
