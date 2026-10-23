package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorkforceReads struct {
	tests        []*worker.WorkerDOTTest
	draw         *worker.DOTRandomDraw
	entries      []*worker.DOTRandomDrawEntry
	injuries     []*worker.WorkerInjury
	cases        []*worker.WorkerLeaveCase
	permits      []*permit.Permit
	requirements []*permit.Requirement

	testsReq    *workerports.ListWorkerDOTTestsRequest
	injuriesReq *workerports.ListWorkerInjuriesRequest
	casesReq    *workerports.ListLeaveCasesRequest
}

func (f *fakeWorkforceReads) ListPools(
	context.Context,
	*workerports.ListDOTRandomPoolsRequest,
) (*pagination.CursorListResult[*worker.DOTRandomPool], error) {
	return &pagination.CursorListResult[*worker.DOTRandomPool]{}, nil
}

func (f *fakeWorkforceReads) ListDraws(
	context.Context,
	*workerports.ListDOTRandomDrawsRequest,
) ([]*worker.DOTRandomDraw, error) {
	return []*worker.DOTRandomDraw{f.draw}, nil
}

func (f *fakeWorkforceReads) GetDraw(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.DOTRandomDraw, error) {
	return f.draw, nil
}

func (f *fakeWorkforceReads) ListDrawEntries(
	context.Context,
	*workerports.ListDOTRandomDrawEntriesRequest,
) ([]*worker.DOTRandomDrawEntry, error) {
	return f.entries, nil
}

func (f *fakeWorkforceReads) ListCases(
	_ context.Context,
	req *workerports.ListLeaveCasesRequest,
) ([]*worker.WorkerLeaveCase, error) {
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

func TestGetDOTRandomDraw_ListsEverySelection(t *testing.T) {
	t.Parallel()

	reads := &fakeWorkforceReads{
		draw: &worker.DOTRandomDraw{
			ID:        pulid.MustNew("drdraw_"),
			PeriodKey: "2026-Q3",
			Status:    worker.RandomDrawStatusFinal,
		},
		entries: []*worker.DOTRandomDrawEntry{{
			ID:        pulid.MustNew("drde_"),
			WorkerID:  pulid.MustNew("wrk_"),
			Substance: worker.DOTSubstanceAlcohol,
			Rank:      1,
			Status:    worker.RandomEntrySelected,
		}},
	}
	tool := newGetDOTRandomDrawTool(reads, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		paramDrawID: reads.draw.ID.String(),
	}))
	require.NoError(t, err)
	detail := result.(*randomDrawDetail)
	assert.Equal(t, "2026-Q3", detail.PeriodKey)
	require.Len(t, detail.Selections, 1)
	assert.Equal(t, "Alcohol", detail.Selections[0].Substance)
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
	for _, tool := range []agentports.AgentQueryTool{
		newListDOTRandomDrawsTool(reads, nil),
		newGetDOTRandomDrawTool(reads, nil),
		newListShipmentPermitsTool(reads, nil),
	} {
		assert.Equal(t, false, tool.ParamSchema()["additionalProperties"], tool.Name())
		policy := tool.Policy()
		assert.Equal(t, permission.OpRead, policy.Operation, tool.Name())
		assert.Equal(t, agent.ToolKindQuery, policy.Kind, tool.Name())
	}
}
