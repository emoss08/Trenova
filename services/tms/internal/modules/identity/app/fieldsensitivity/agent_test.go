package fieldsensitivity

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAgentCeiling(t *testing.T) {
	t.Parallel()

	person := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
	}
	agent := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agd_"),
		OrganizationID: pulid.MustNew("org_"),
	}
	restrictedRole := &fakeEngine{levels: map[string]permission.FieldSensitivity{
		permission.ResourceWorker.String(): permission.SensitivityRestricted,
	}}

	tests := []struct {
		name    string
		actor   *permissionports.RequestActor
		granted permission.FieldSensitivity
		want    permission.FieldSensitivity
	}{
		{"an agent with no data access reads at internal", agent, "",
			permission.SensitivityInternal},
		{"an agent reads at its data access", agent, permission.SensitivityRestricted,
			permission.SensitivityRestricted},
		{"an agent's confidential data access reads at restricted", agent,
			permission.SensitivityConfidential, permission.SensitivityRestricted},
		{"a person reads at their role with no agent limit", person, "",
			permission.SensitivityRestricted},
		{"a person is lowered to the agent's data access", person,
			permission.SensitivityInternal, permission.SensitivityInternal},
		{"no actor reads at internal", nil, "", permission.SensitivityInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, AgentCeiling(
				t.Context(), restrictedRole, tt.actor, tt.granted, permission.ResourceWorker,
			))
		})
	}
}

func TestAgentCeilings_AsksOncePerResource(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{levels: map[string]permission.FieldSensitivity{
		permission.ResourceWorker.String(): permission.SensitivityRestricted,
	}}
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
	}
	ceilings := NewAgentCeilings(engine, actor, "")

	for range 3 {
		assert.Equal(t, permission.SensitivityRestricted,
			ceilings.For(t.Context(), permission.ResourceWorker))
	}
	assert.Equal(t, int32(1), engine.calls.Load())
}
