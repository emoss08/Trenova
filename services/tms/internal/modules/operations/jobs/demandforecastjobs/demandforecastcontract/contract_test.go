package demandforecastcontract

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

var tenantInfo = pagination.TenantInfo{
	OrgID:  pulid.ID("org_a"),
	BuID:   pulid.ID("bu_a"),
	UserID: pulid.ID("usr_a"),
}

type startedRun struct {
	client.WorkflowRun
}

func TestStartFitting_StartsOneFitPerTenantOnTheDispatchQueue(t *testing.T) {
	t.Parallel()

	workflows := kernelportsmock.NewMockWorkflowStarter(t)
	var options client.StartWorkflowOptions
	var args []any
	workflows.EXPECT().
		StartWorkflow(mock.Anything, mock.Anything, FitTenantWorkflowName, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			got client.StartWorkflowOptions,
			_ any,
			payload ...any,
		) (client.WorkflowRun, error) {
			options, args = got, payload
			return startedRun{}, nil
		}).
		Once()

	err := NewStarter(StarterParams{Workflows: workflows}).StartFitting(t.Context(), tenantInfo)
	require.NoError(t, err)

	assert.Equal(t, "demand-forecast-fit-org_a-bu_a", options.ID)
	assert.Equal(t, FitWorkflowID(tenantInfo), options.ID)
	assert.Equal(t, temporaltype.TaskQueueDispatch.String(), options.TaskQueue)
	assert.Equal(t,
		enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		options.WorkflowIDConflictPolicy,
		"turning the setting on twice while a fit runs joins the running fit",
	)
	assert.Equal(t, "org_a", options.Priority.FairnessKey)
	assert.Equal(t, []any{&FitPayload{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}}, args)
}

func TestFitWorkflowID_IsTheSameWhoeverAsks(t *testing.T) {
	t.Parallel()

	other := tenantInfo
	other.UserID = pulid.ID("usr_b")

	assert.Equal(t, FitWorkflowID(tenantInfo), FitWorkflowID(other))
	assert.NotEqual(t, FitWorkflowID(tenantInfo), FitWorkflowID(pagination.TenantInfo{
		OrgID: tenantInfo.OrgID,
		BuID:  pulid.ID("bu_b"),
	}))
}

func TestStartFitting_WrapsTheStartError(t *testing.T) {
	t.Parallel()

	temporalDown := errors.New("temporal down")
	workflows := kernelportsmock.NewMockWorkflowStarter(t)
	workflows.EXPECT().
		StartWorkflow(mock.Anything, mock.Anything, FitTenantWorkflowName, mock.Anything).
		Return(nil, temporalDown).
		Once()

	err := NewStarter(StarterParams{Workflows: workflows}).StartFitting(t.Context(), tenantInfo)

	require.ErrorIs(t, err, temporalDown)
	assert.EqualError(t, err, "start fitting the freight forecast: temporal down")
}
