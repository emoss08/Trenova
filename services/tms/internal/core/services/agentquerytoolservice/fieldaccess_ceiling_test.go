package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func agentParams(
	params map[string]any,
	dataAccess permission.FieldSensitivity,
) *serviceports.QueryToolParams {
	out := testParams(params)
	out.Actor.PrincipalType = serviceports.PrincipalTypeAgent
	out.Actor.PrincipalID = pulid.MustNew("agdef_")
	out.Actor.UserID = pulid.Nil
	out.DataAccessCeiling = dataAccess

	return out
}

func chatParams(
	params map[string]any,
	dataAccess permission.FieldSensitivity,
) *serviceports.QueryToolParams {
	out := testParams(params)
	out.Actor.PrincipalType = serviceports.PrincipalTypeUser
	out.Actor.PrincipalID = out.Actor.UserID
	out.DataAccessCeiling = dataAccess

	return out
}

func personReaching(
	resource permission.Resource,
	sensitivity permission.FieldSensitivity,
) *fakePermissions {
	return &fakePermissions{
		readable: map[string]*serviceports.ResourcePermissionDetail{
			resource.String(): {
				Resource:       resource.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: sensitivity,
			},
		},
	}
}

func TestCeiling_AnUnattendedAgentReadsAtItsOwnDataAccess(t *testing.T) {
	t.Parallel()

	access := newFieldAccess(&fakePermissions{})
	resource := permission.ResourceInvoice

	assert.Equal(t, permission.SensitivityRestricted, access.ceiling(t.Context(),
		agentParams(nil, permission.SensitivityRestricted), resource),
		"an agent set to Restricted sees amounts with nobody watching")
	assert.Equal(t, permission.SensitivityInternal, access.ceiling(t.Context(),
		agentParams(nil, permission.SensitivityInternal), resource))
	assert.Equal(t, permission.SensitivityInternal, access.ceiling(t.Context(),
		agentParams(nil, ""), resource), "an agent with no setting reads at Internal")
	assert.Equal(t, permission.SensitivityRestricted, access.ceiling(t.Context(),
		agentParams(nil, permission.SensitivityConfidential), resource),
		"nothing above Restricted reaches a model, whatever the setting says")
}

func TestCeiling_AChatRunReadsAtTheLowerOfThePersonAndTheAgent(t *testing.T) {
	t.Parallel()

	resource := permission.ResourceInvoice
	restrictedPerson := newFieldAccess(
		personReaching(resource, permission.SensitivityRestricted),
	)

	assert.Equal(t, permission.SensitivityInternal, restrictedPerson.ceiling(t.Context(),
		chatParams(nil, permission.SensitivityInternal), resource),
		"an agent set to Internal keeps amounts from a person who could see them")
	assert.Equal(t, permission.SensitivityRestricted, restrictedPerson.ceiling(t.Context(),
		chatParams(nil, permission.SensitivityRestricted), resource))
	assert.Equal(t, permission.SensitivityRestricted, restrictedPerson.ceiling(t.Context(),
		chatParams(nil, ""), resource), "with no agent, the person's own access applies")
}

func TestCeiling_NeverReadsAboveThePerson(t *testing.T) {
	t.Parallel()

	resource := permission.ResourceInvoice
	internalPerson := newFieldAccess(personReaching(resource, permission.SensitivityInternal))

	assert.Equal(t, permission.SensitivityInternal, internalPerson.ceiling(t.Context(),
		chatParams(nil, permission.SensitivityRestricted), resource),
		"an agent set to Restricted cannot lift a person past their own role")
}
