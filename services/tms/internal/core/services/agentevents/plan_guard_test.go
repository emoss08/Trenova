package agentevents_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevents"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPublishSkipsOrganizationsWithoutAutomatedAgents(t *testing.T) {
	t.Parallel()

	definitions := &fakeDefinitions{items: []*agentdefinition.Definition{{ID: pulid.MustNew("agd_")}}}
	runs := &fakeRunService{}
	publisher := agentevents.New(agentevents.Params{
		Logger:      zap.NewNop(),
		Definitions: definitions,
		RunService:  runs,
		Plans:       plantest.Restricting(t, platformplan.CapabilityAgentAutomation),
	})

	publisher.Publish(t.Context(), services.AgentEvent{
		Kind:       agent.EventShipmentCreated,
		SubjectID:  pulid.MustNew("shp_"),
		TenantInfo: plantest.Tenant(),
	})

	assert.Empty(t, runs.started)
	assert.Empty(t, definitions.lastReq.EventKind)
}
