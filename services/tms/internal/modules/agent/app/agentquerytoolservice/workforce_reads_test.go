package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerinjury"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerdotports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerleaveports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workersafetyports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorkforceReads struct {
	tests        []*workerdot.WorkerDOTTest
	injuries     []*workerinjury.WorkerInjury
	cases        []*workerleave.WorkerLeaveCase
	permits      []*permit.Permit
	requirements []*permit.Requirement

	testsReq    *workerdotports.ListWorkerDOTTestsRequest
	injuriesReq *workersafetyports.ListWorkerInjuriesRequest
	casesReq    *workerleaveports.ListLeaveCasesRequest
}

func (f *fakeWorkforceReads) ListCases(
	_ context.Context,
	req *workerleaveports.ListLeaveCasesRequest,
) ([]*workerleave.WorkerLeaveCase, error) {
	f.casesReq = req
	return f.cases, nil
}

func (f *fakeWorkforceReads) ListPermits(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*permit.Permit, error) {
	return f.permits, nil
}

func (f *fakeWorkforceReads) ListRequirements(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*permit.Requirement, error) {
	return f.requirements, nil
}

func TestListShipmentPermits_PairsRequirementsWithPermits(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")
	permitID := pulid.MustNew("pmt_")
	expires := int64(1_791_000_000)
	reads := &fakeWorkforceReads{
		requirements: []*permit.Requirement{{
			ID:                  pulid.MustNew("prq_"),
			StateID:             stateID,
			Status:              permit.RequirementSatisfied,
			State:               &usstate.UsState{Abbreviation: "IN"},
			SatisfiedByPermitID: &permitID,
		}},
		permits: []*permit.Permit{{
			ID:           permitID,
			StateID:      stateID,
			PermitNumber: "IN-44120",
			Status:       permit.StatusActive,
			ExpiresAt:    &expires,
		}},
	}
	tool := newListShipmentPermitsTool(reads, &fakePermissions{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		paramShipmentID: pulid.MustNew("shp_").String(),
	}))
	require.NoError(t, err)
	permits := result.(*shipmentPermits)
	require.Len(t, permits.Requirements, 1)
	assert.Equal(t, "IN", permits.Requirements[0].State)
	assert.Equal(t, permitID.String(), permits.Requirements[0].SatisfiedByPermitID)
	require.Len(t, permits.Permits, 1)
	assert.Equal(t, stateID.String(), permits.Permits[0].StateID)
	assert.Equal(t, permission.ResourcePermit, tool.Policy().Resource)
}

func TestWorkforceReads_AreClosedReadsOfTheirResource(t *testing.T) {
	t.Parallel()

	reads := &fakeWorkforceReads{}
	for _, tool := range []agenttoolports.AgentQueryTool{
		newListShipmentPermitsTool(reads, nil),
	} {
		assert.Equal(t, false, tool.ParamSchema()["additionalProperties"], tool.Name())
		policy := tool.Policy()
		assert.Equal(t, permission.OpRead, policy.Operation, tool.Name())
		assert.Equal(t, agent.ToolKindQuery, policy.Kind, tool.Name())
	}
}
