package agentcontrolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type promoter struct {
	services.AgentTrustService
	asked []*services.PromoteReadyRequest
}

func (p *promoter) PromoteReady(
	_ context.Context,
	req *services.PromoteReadyRequest,
) ([]services.ToolPromotion, error) {
	p.asked = append(p.asked, req)
	return nil, nil
}

func intPtr(v int) *int { return &v }

func TestTurningEarnedAutonomyOnPromotesWhatAlreadyEarnedIt(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(defaultControl())
	trust := &promoter{}
	svc.trust = trust
	user := pulid.ID("usr_admin")

	_, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		EarnedAutonomy: boolPtr(true),
	}, userActor(user))

	require.NoError(t, err)
	require.Len(t, trust.asked, 1)
	assert.Equal(t, 10, trust.asked[0].Threshold)
	assert.Equal(t, user, trust.asked[0].DecidedBy)
}

func TestLoweringTheThresholdPromotesButRaisingItDoesNot(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.EarnedAutonomy = true
	svc, _ := newTestService(control)
	trust := &promoter{}
	svc.trust = trust

	_, err := svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		PromotionThreshold: intPtr(25),
	}, userActor(pulid.ID("usr_admin")))
	require.NoError(t, err)
	assert.Empty(t, trust.asked)

	_, err = svc.Update(t.Context(), &services.UpdateAgentControlRequest{
		PromotionThreshold: intPtr(5),
	}, userActor(pulid.ID("usr_admin")))
	require.NoError(t, err)
	require.Len(t, trust.asked, 1)
	assert.Equal(t, 5, trust.asked[0].Threshold)
}
