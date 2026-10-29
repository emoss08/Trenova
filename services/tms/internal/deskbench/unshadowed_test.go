package deskbench

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type storedControl struct {
	control *agentcontrol.AgentControl
	updated *agentcontrol.AgentControl
}

func (s *storedControl) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*agentcontrol.AgentControl, error) {
	return s.control, nil
}

func (s *storedControl) Update(
	_ context.Context,
	entity *agentcontrol.AgentControl,
) (*agentcontrol.AgentControl, error) {
	s.updated = entity
	return entity, nil
}

func TestUnshadowed_RunsLiveWhateverTheOrganizationsSwitch(t *testing.T) {
	t.Parallel()

	stored := &storedControl{control: &agentcontrol.AgentControl{
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		ShadowMode:         true,
		PromotionThreshold: 7,
	}}
	repo := unshadowed{AgentControlRepository: stored}

	control, err := repo.GetOrCreate(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.False(t, control.ShadowMode)
	require.Equal(t, 7, control.PromotionThreshold)
	require.True(t, stored.control.ShadowMode, "the stored control is not changed")

	control.PromotionThreshold = 9
	_, err = repo.Update(t.Context(), control)
	require.NoError(t, err)
	require.True(t, stored.updated.ShadowMode, "an update keeps the person's shadow switch")
	require.Equal(t, 9, stored.updated.PromotionThreshold)
}
