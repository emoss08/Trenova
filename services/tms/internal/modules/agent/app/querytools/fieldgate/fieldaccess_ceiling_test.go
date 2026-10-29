package fieldgate

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func TestCeiling_AnUnattendedAgentReadsAtItsOwnDataAccess(t *testing.T) {
	t.Parallel()

	access := NewAccess(&querytooltest.FakePermissions{})
	resource := permission.ResourceInvoice

	assert.Equal(t, permission.SensitivityRestricted, access.ceiling(t.Context(),
		querytooltest.AgentParams(nil, permission.SensitivityRestricted), resource),
		"an agent set to Restricted sees amounts with nobody watching")
	assert.Equal(t, permission.SensitivityInternal, access.ceiling(t.Context(),
		querytooltest.AgentParams(nil, permission.SensitivityInternal), resource))
	assert.Equal(t, permission.SensitivityInternal, access.ceiling(t.Context(),
		querytooltest.AgentParams(nil, ""), resource), "an agent with no setting reads at Internal")
	assert.Equal(t, permission.SensitivityRestricted, access.ceiling(t.Context(),
		querytooltest.AgentParams(nil, permission.SensitivityConfidential), resource),
		"nothing above Restricted reaches a model, whatever the setting says")
}

func TestCeiling_AChatRunReadsAtTheLowerOfThePersonAndTheAgent(t *testing.T) {
	t.Parallel()

	resource := permission.ResourceInvoice
	restrictedPerson := NewAccess(
		querytooltest.PersonReaching(resource, permission.SensitivityRestricted),
	)

	assert.Equal(t, permission.SensitivityInternal, restrictedPerson.ceiling(t.Context(),
		querytooltest.ChatParams(nil, permission.SensitivityInternal), resource),
		"an agent set to Internal keeps amounts from a person who could see them")
	assert.Equal(t, permission.SensitivityRestricted, restrictedPerson.ceiling(t.Context(),
		querytooltest.ChatParams(nil, permission.SensitivityRestricted), resource))
	assert.Equal(t, permission.SensitivityRestricted, restrictedPerson.ceiling(
		t.Context(),
		querytooltest.ChatParams(
			nil,
			"",
		),
		resource,
	), "with no agent, the person's own access applies")
}

func TestCeiling_NeverReadsAboveThePerson(t *testing.T) {
	t.Parallel()

	resource := permission.ResourceInvoice
	internalPerson := NewAccess(
		querytooltest.PersonReaching(resource, permission.SensitivityInternal),
	)

	assert.Equal(t, permission.SensitivityInternal, internalPerson.ceiling(t.Context(),
		querytooltest.ChatParams(nil, permission.SensitivityRestricted), resource),
		"an agent set to Restricted cannot lift a person past their own role")
}
