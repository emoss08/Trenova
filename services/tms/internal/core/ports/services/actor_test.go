package services

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func actorOf(principal PrincipalType) *RequestActor {
	return &RequestActor{
		PrincipalType:  principal,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestRequestActor_PersonUserIDNamesOnlyAPerson(t *testing.T) {
	t.Parallel()

	person := actorOf(PrincipalTypeUser)
	assert.Equal(t, person.UserID, person.PersonUserID())

	for _, principal := range []PrincipalType{
		PrincipalTypeAgent,
		PrincipalTypeAPIKey,
		PrincipalTypeSystem,
	} {
		assert.Equal(t, pulid.Nil, actorOf(principal).PersonUserID(), principal)
	}

	unnamed := actorOf("")
	assert.Equal(t, unnamed.UserID, unnamed.PersonUserID(), "an actor with only a user is that user")
	unnamed.APIKeyID = pulid.MustNew("ak_")
	assert.Equal(t, pulid.Nil, unnamed.PersonUserID())

	var missing *RequestActor
	assert.Equal(t, pulid.Nil, missing.PersonUserID())
}

func TestRequestActor_AnAgentsDatabaseScopeNamesNoUser(t *testing.T) {
	t.Parallel()

	agent := actorOf(PrincipalTypeAgent)
	scope := agent.DBTenant()
	assert.Equal(t, agent.OrganizationID, scope.OrganizationID)
	assert.Equal(t, agent.BusinessUnitID, scope.BusinessUnitID)
	assert.Equal(t, pulid.Nil, scope.UserID,
		"the system user an unattended run is attributed to lends its name, not its reach")
	assert.Equal(t, agent.UserID, agent.TenantInfo().UserID,
		"attribution still reads the system user from the tenant")

	person := actorOf(PrincipalTypeUser)
	assert.Equal(t, person.UserID, person.DBTenant().UserID)

	carried := struct{ Actor *RequestActor }{Actor: agent}
	found, ok := dbscope.TenantOf(carried)
	require.True(t, ok)
	assert.Equal(t, pulid.Nil, found.UserID)
	assert.Equal(t, agent.OrganizationID, found.OrganizationID)
}
