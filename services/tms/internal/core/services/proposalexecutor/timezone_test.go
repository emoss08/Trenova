package proposalexecutor

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubZones struct {
	zone string
	err  error
	asks []pagination.TenantInfo
}

func (z *stubZones) TenantTimezone(_ context.Context, info pagination.TenantInfo) (string, error) {
	z.asks = append(z.asks, info)

	return z.zone, z.err
}

type zoneTool struct {
	recordingTool

	validatedIn string
}

func (t *zoneTool) Validate(_ context.Context, params services.ToolExecuteParams) error {
	t.validatedIn = params.Timezone

	return nil
}

func TestExecute_HandsTheToolTheTenantsTimezone(t *testing.T) {
	t.Parallel()

	tool := &zoneTool{recordingTool: recordingTool{
		name:      "create_shipment",
		resource:  permission.ResourceShipment,
		operation: permission.OpCreate,
	}}
	proposal := testProposal(tool.name, map[string]any{})
	actor := testActor(proposal.OrganizationID, proposal.BusinessUnitID)
	zones := &stubZones{zone: "America/Chicago"}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})
	executor.zones = zones

	require.NoError(t, executor.Execute(t.Context(), proposal, nil, actor))

	assert.Equal(t, "America/Chicago", tool.lastParams.Timezone)
	assert.Equal(t, "America/Chicago", tool.validatedIn)
	require.NotEmpty(t, zones.asks)
	assert.Equal(t, pagination.TenantInfo{
		OrgID:  proposal.OrganizationID,
		BuID:   proposal.BusinessUnitID,
		UserID: actor.UserID,
	}, zones.asks[0])
}

func TestExecute_RefusesWhenTheTimezoneCannotBeRead(t *testing.T) {
	t.Parallel()

	tool := &zoneTool{recordingTool: recordingTool{
		name:      "create_shipment",
		resource:  permission.ResourceShipment,
		operation: permission.OpCreate,
	}}
	proposal := testProposal(tool.name, map[string]any{})
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})
	executor.zones = &stubZones{err: errors.New("database is down")}

	err := executor.Execute(
		t.Context(), proposal, nil, testActor(proposal.OrganizationID, proposal.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Zero(t, tool.calls, "a local time is never read in a zone that was guessed")
}
