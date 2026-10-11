package accountingsynccontract

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

func TestDriftWorkflowIDIsPerConnection(t *testing.T) {
	t.Parallel()
	id := pulid.MustNew("acctc_")
	assert.Equal(t, "accounting-drift:"+id.String(), DriftWorkflowID(id))
}

type fakeStarter struct {
	kernelports.WorkflowStarter
	options  []client.StartWorkflowOptions
	payloads []any
	err      error
}

func (f *fakeStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	f.options = append(f.options, options)
	f.payloads = append(f.payloads, args...)
	return nil, f.err
}

func TestCheckNowStartsOneRunPerConnection(t *testing.T) {
	t.Parallel()
	starter := &fakeStarter{}
	checker := NewDriftChecker(DriftCheckerParams{Workflows: starter})
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	connectionID := pulid.MustNew("acctc_")

	require.NoError(t, checker.CheckNow(t.Context(), tenant, connectionID))

	require.Len(t, starter.options, 1)
	assert.Equal(t, DriftWorkflowID(connectionID), starter.options[0].ID)
	assert.Equal(t, enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		starter.options[0].WorkflowIDConflictPolicy, "a second click joins the run already going")
	payload, ok := starter.payloads[0].(*DriftPayload)
	require.True(t, ok)
	assert.Equal(t, connectionID, payload.ConnectionID)
	assert.Equal(t, tenant.OrgID, payload.OrganizationID)

	starter.err = kernelports.ErrWorkflowStarterDisabled
	err := checker.CheckNow(t.Context(), tenant, connectionID)
	assert.True(t, errortypes.IsBusinessError(err))
}
