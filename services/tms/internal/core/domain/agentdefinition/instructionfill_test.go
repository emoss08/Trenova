package agentdefinition_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_FillsInstructionVariables(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 7, 23, 30, 0, 0, time.UTC).Unix()
	prompt := definitionWithInstructions(
		"You are the detention desk for {{organization}}. Ask {{ user.name }} ({{user.role}}) " +
			"before {{today}} ends. Leave {{shipment.id}} alone.",
	).BuildSystemPrompt(agentdefinition.RuntimeContext{
		OrganizationName: "Acme Freight",
		Timezone:         "America/Chicago",
		Now:              now,
		User:             &agentdefinition.RuntimeUser{Name: "Sarah Alvarez", Roles: []string{"Dispatch lead", "Owner"}},
	})

	assert.Contains(t, prompt, "You are the detention desk for Acme Freight. Ask Sarah Alvarez "+
		"(Dispatch lead, Owner) before Wednesday, October 7, 2026 ends. Leave {{shipment.id}} alone.")
}

func TestBuildSystemPrompt_FillsInstructionVariablesWithoutAPerson(t *testing.T) {
	t.Parallel()

	prompt := definitionWithInstructions("Send {{user.name}} what {{organization}} needs {{today}}, whatever {{user.role}}.").
		BuildSystemPrompt(agentdefinition.RuntimeContext{})

	assert.Contains(t, prompt, "Send the person asking what your organization needs today, whatever their role.")
}
