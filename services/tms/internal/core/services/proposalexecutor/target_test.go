package proposalexecutor

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeVersions struct {
	version int64
	err     error
	asked   []services.ToolTarget
}

func (f *fakeVersions) Version(
	_ context.Context,
	_ pagination.TenantInfo,
	target services.ToolTarget,
) (int64, error) {
	f.asked = append(f.asked, target)

	return f.version, f.err
}

func pinnedProposal(tool string, version int64) *agent.AgentProposal {
	proposal := testProposal(tool, map[string]any{"shipmentId": "shp_1"})
	proposal.TargetResource = string(permission.ResourceShipment)
	proposal.TargetID = pulid.MustNew("shp_")
	proposal.TargetVersion = version

	return proposal
}

// A hold proposed on a shipment that has since been delivered is not the
// change anyone approved. The version pinned at proposal time is compared
// with the record's before the tool runs, and a different one refuses.
func TestExecute_RefusesWhenTheRecordChangedSinceTheProposal(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "place_shipment_hold",
		resource:  permission.ResourceShipment,
		operation: permission.OpUpdate,
	}
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})
	executor.versions = &fakeVersions{version: 5}

	proposal := pinnedProposal("place_shipment_hold", 3)
	err := executor.Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)

	require.ErrorIs(t, err, ErrTargetChanged)
	assert.Contains(t, err.Error(), "version 5")
	assert.Contains(t, err.Error(), "was at 3")
	assert.Zero(t, tool.calls, "the tool never runs against a record that moved on")
	require.Len(t, repo.recorded, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, repo.recorded[0].status)
}

func TestExecute_RunsWhenTheRecordIsUnchanged(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "place_shipment_hold",
		resource:  permission.ResourceShipment,
		operation: permission.OpUpdate,
	}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	versions := &fakeVersions{version: 3}
	executor.versions = versions

	proposal := pinnedProposal("place_shipment_hold", 3)
	require.NoError(
		t,
		executor.Execute(
			t.Context(),
			proposal,
			nil,
			testActor(proposal.OrganizationID, proposal.BusinessUnitID),
		),
	)

	assert.Equal(t, 1, tool.calls)
	require.Len(t, versions.asked, 2, "the version is checked before the write and read after it")
	assert.Equal(t, proposal.TargetID, versions.asked[0].ID)
	assert.Equal(t, proposal.TargetID, versions.asked[1].ID)
}

// A record that cannot be read is not equal to anything: a proposal against a
// deleted shipment must not run because the comparison could not happen.
func TestExecute_RefusesWhenTheRecordCannotBeRead(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "place_shipment_hold",
		resource:  permission.ResourceShipment,
		operation: permission.OpUpdate,
	}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	executor.versions = &fakeVersions{err: errors.New("no rows")}

	proposal := pinnedProposal("place_shipment_hold", 3)
	err := executor.Execute(
		t.Context(),
		proposal,
		nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)

	require.ErrorIs(t, err, ErrTargetChanged)
	assert.Zero(t, tool.calls)
}

// An older proposal has no pin, and a tool with no single target leaves none.
// Neither is refused for it: the check only ever fires on a pin that no
// longer matches.
func TestExecute_RunsAnUnpinnedProposalUnchecked(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "raise_exception",
		resource:  permission.ResourceShipment,
		operation: permission.OpUpdate,
	}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	versions := &fakeVersions{version: 99}
	executor.versions = versions

	proposal := testProposal("raise_exception", map[string]any{})
	require.NoError(
		t,
		executor.Execute(
			t.Context(),
			proposal,
			nil,
			testActor(proposal.OrganizationID, proposal.BusinessUnitID),
		),
	)

	assert.Equal(t, 1, tool.calls)
	assert.Empty(t, versions.asked, "nothing to compare, nothing asked")
}
