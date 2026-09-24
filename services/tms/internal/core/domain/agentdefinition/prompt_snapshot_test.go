package agentdefinition_test

import (
	"flag"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

var updatePrompts = flag.Bool("update", false, "rewrite the prompt snapshots")

const (
	promptSnapshotDir = "testdata/prompts"
	snapshotNow       = int64(1789560000)
	parentAgentName   = "Dispatch assistant"
)

var (
	snapshotOrgID    = pulid.ID("org_01JSNAPSHOT0000000000000000")
	snapshotBuID     = pulid.ID("bu_01JSNAPSHOT00000000000000000")
	snapshotUserID   = pulid.ID("usr_01JSNAPSHOT0000000000000000")
	snapshotAgentID  = pulid.ID("agdef_01JSNAPSHOT00000000000000")
	snapshotThreadID = pulid.ID("athr_01JSNAPSHOT000000000000000")
	snapshotRunID    = pulid.ID("arun_01JSNAPSHOT000000000000000")
	snapshotParentID = pulid.ID("agdef_01JSNAPSHOTPARENT0000000")
)

type promptContext struct {
	name    string
	request func(definition *agentdefinition.Definition) *serviceports.RunRequest
}

func snapshotDefinition(template agentdefinition.Template) *agentdefinition.Definition {
	output := agentdefinition.OutputReport
	if template.StarterTrigger() == agentdefinition.TriggerChat {
		output = agentdefinition.OutputConversational
	}

	definition := &agentdefinition.Definition{
		ID:              snapshotAgentID,
		OrganizationID:  snapshotOrgID,
		BusinessUnitID:  snapshotBuID,
		Name:            template.Label(),
		Description:     template.Description(),
		Template:        template,
		Instructions:    template.StarterInstructions(),
		ToolNames:       template.StarterTools(),
		AutonomyCeiling: template.StarterCeiling(),
		TriggerMode:     template.StarterTrigger(),
		EventKinds:      template.StarterEvents(),
		CronExpression:  template.StarterCron(),
		OutputMode:      output,
		Enabled:         true,
	}
	definition.ApplyDefaults()

	return definition
}

func snapshotUser() *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    snapshotUserID,
		UserID:         snapshotUserID,
		OrganizationID: snapshotOrgID,
		BusinessUnitID: snapshotBuID,
	}
}

func canonicalContext(trigger agent.RunTrigger) agentdefinition.RuntimeContext {
	return agentdefinition.RuntimeContext{
		OrganizationName: "Acme Freight",
		BusinessUnitName: "West Coast",
		Timezone:         "America/Chicago",
		Now:              snapshotNow,
		Trigger:          trigger,
	}
}

func promptContexts() []promptContext {
	return []promptContext{
		{
			name: "chat",
			request: func(definition *agentdefinition.Definition) *serviceports.RunRequest {
				rc := canonicalContext(agent.RunTriggerChat)
				rc.User = &agentdefinition.RuntimeUser{
					Name:  "Maria Ortiz",
					Email: "maria@acme.example",
					Roles: []string{"Dispatcher"},
				}
				rc.Page = &agentdefinition.PageContext{
					Path:  "/shipments",
					Title: "Shipments",
				}
				rc.Guide = true

				return &serviceports.RunRequest{
					Definition: definition,
					Actor:      snapshotUser(),
					Context:    rc,
					ThreadID:   snapshotThreadID,
					Publishes:  true,
					Input:      "What needs my attention today?",
				}
			},
		},
		{
			name: "background",
			request: func(definition *agentdefinition.Definition) *serviceports.RunRequest {
				rc := canonicalContext(agent.RunTriggerEvent)
				rc.Subject = &agentdefinition.RuntimeSubject{
					Type:  agent.SubjectShipmentMove,
					ID:    "smv_01JSNAPSHOT000000000000000",
					Label: "Move 1 of shipment S-1001 for Acme Freight",
					Notes: "Picks up in Dallas tomorrow at 08:00; no driver assigned.",
				}
				actor := snapshotUser()
				actor.PrincipalType = serviceports.PrincipalTypeAgent
				actor.PrincipalID = snapshotAgentID

				return &serviceports.RunRequest{
					Definition: definition,
					Actor:      actor,
					Context:    rc,
					RunID:      snapshotRunID,
					Unattended: true,
					Input:      "Work the record this run is about.",
				}
			},
		},
		{
			name: "delegated",
			request: func(definition *agentdefinition.Definition) *serviceports.RunRequest {
				rc := canonicalContext(agent.RunTriggerChat)
				rc.DelegatedBy = parentAgentName

				return &serviceports.RunRequest{
					Definition: definition,
					Actor:      snapshotUser(),
					Context:    rc,
					ThreadID:   snapshotThreadID,
					Publishes:  true,
					Input: "Find the shipments for Acme Freight that picked up this week " +
						"and hand back their ids.",
					Delegation: &serviceports.Delegation{
						ParentAgentID:   snapshotParentID,
						ParentAgentName: parentAgentName,
						CallID:          "call_delegate_snapshot",
						StepScope:       "snapshot",
					},
				}
			},
		},
	}
}

func snapshotRuntime(t *testing.T) *agentruntime.Service {
	t.Helper()

	kit, err := agentevalgate.NewKit()
	require.NoError(t, err)

	return kit.NewRuntime(agentevalgate.RuntimeParams{
		Completion:  &agentruntimetest.ScriptedCompletion{},
		Permissions: &agentruntimetest.StubPermissions{},
	})
}

func TestPromptSnapshots(t *testing.T) {
	t.Parallel()

	rt := snapshotRuntime(t)
	for _, template := range agentdefinition.AllTemplates() {
		for _, pc := range promptContexts() {
			name := fmt.Sprintf("%s_%s", template, pc.name)
			t.Run(name, func(t *testing.T) {
				turn := rt.OpenTurn(t.Context(), pc.request(snapshotDefinition(template)))
				system := turn.State().System + "\n"

				agentevalgate.Golden(t,
					filepath.Join(promptSnapshotDir, name+".golden"),
					[]byte(system),
					*updatePrompts,
					"go test -tags nofitz -run TestPromptSnapshots "+
						"./internal/core/domain/agentdefinition/ -update",
				)
			})
		}
	}
}
