package agentevents_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentevents"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPublishSkipsOrganizationsWithoutAutomatedAgents(t *testing.T) {
	t.Parallel()

	definitions := &fakeDefinitions{
		items: []*agentdefinition.Definition{{ID: pulid.MustNew("agd_")}},
	}
	runs := &fakeRunService{}
	publisher := agentevents.New(agentevents.Params{
		Logger:      zap.NewNop(),
		Definitions: definitions,
		RunService:  runs,
		Plans:       plantest.Restricting(t, platformplan.CapabilityAgentAutomation),
	})

	publisher.Publish(t.Context(), agentports.AgentEvent{
		Kind:       agenttrigger.EventShipmentCreated,
		SubjectID:  pulid.MustNew("shp_"),
		TenantInfo: plantest.Tenant(),
	})

	assert.Empty(t, runs.started)
	assert.Empty(t, definitions.lastReq.EventKind)
}
