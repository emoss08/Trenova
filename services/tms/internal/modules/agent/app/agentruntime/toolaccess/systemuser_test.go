package toolaccess

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestTurnAttribution_UsageIsNotChargedToTheSystemUser(t *testing.T) {
	t.Parallel()

	unattended := TurnAttribution(&agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.SystemUserActor(),
	})
	assert.Equal(t, pulid.Nil, unattended.UserID)

	actor := runtimetest.Actor()
	person := TurnAttribution(
		&agentruntimeports.RunRequest{Definition: runtimetest.Definition(), Actor: actor},
	)
	assert.Equal(t, actor.UserID, person.UserID)
}

func TestTurnAttribution_ATurnAWaitStartedReadsOnItsOwnLine(t *testing.T) {
	t.Parallel()

	asked := TurnAttribution(
		&agentruntimeports.RunRequest{
			Definition: runtimetest.Definition(),
			Actor:      runtimetest.Actor(),
		},
	)
	assert.Equal(t, aiusage.FeatureAgentTurn, asked.Feature)

	woken := TurnAttribution(&agentruntimeports.RunRequest{
		Definition:   runtimetest.Definition(),
		Actor:        runtimetest.Actor(),
		UsageFeature: aiusage.FeatureAgentWakeUp,
	})
	assert.Equal(t, aiusage.FeatureAgentWakeUp, woken.Feature)
}
