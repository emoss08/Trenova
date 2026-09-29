package proposalpreviewservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubZones struct {
	zone string
	err  error
}

func (z stubZones) TenantTimezone(context.Context, pagination.TenantInfo) (string, error) {
	return z.zone, z.err
}

func TestForProposal_PreviewsInTheTenantsTimezone(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.svc.zones = stubZones{zone: "America/Chicago"}

	_, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: f.proposal(f.cancelParams()),
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	require.NotEmpty(t, f.tool.asked)
	assert.Equal(t, "America/Chicago", f.tool.asked[0].Timezone,
		"the preview reads a local time where the write will")
}

func TestForProposal_SaysWhenTheTimezoneCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.svc.zones = stubZones{err: errors.New("database is down")}

	_, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: f.proposal(f.cancelParams()),
		Viewer:   f.viewer(),
	})

	require.Error(t, err)
	assert.Empty(t, f.tool.asked)
}
