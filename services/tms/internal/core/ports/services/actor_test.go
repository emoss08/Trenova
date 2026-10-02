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

func TestRequestActor_ExecutorUserIDNamesThePersonOrTheSystemUserAnAgentCarries(t *testing.T) {
	t.Parallel()

	person := actorOf(PrincipalTypeUser)
	assert.Equal(t, person.UserID, person.ExecutorUserID())

	unattended := actorOf(PrincipalTypeAgent)
	assert.Equal(t, unattended.UserID, unattended.ExecutorUserID(),
		"an unattended run's automatic writes are executed by the system user it carries")

	bare := actorOf(PrincipalTypeAgent)
	bare.UserID = pulid.Nil
	assert.Equal(t, pulid.Nil, bare.ExecutorUserID())

	for _, principal := range []PrincipalType{"", PrincipalTypeAPIKey, PrincipalTypeSystem} {
		assert.Equal(t, pulid.Nil, actorOf(principal).ExecutorUserID(), principal)
	}

	var missing *RequestActor
	assert.Equal(t, pulid.Nil, missing.ExecutorUserID())
}

func TestRequestActor_AnAgentsAuditActorKeepsTheSystemUserItCarries(t *testing.T) {
	t.Parallel()

	agent := actorOf(PrincipalTypeAgent)
	agent.APIKeyID = pulid.MustNew("ak_")

	audited := agent.AuditActor()
	assert.Equal(t, PrincipalTypeAgent, audited.PrincipalType)
	assert.Equal(t, agent.PrincipalID, audited.PrincipalID, "the principal stays the agent")
	assert.Equal(t, agent.UserID, audited.UserID, "the row names the system user")
	assert.Equal(t, pulid.Nil, audited.APIKeyID)

	unnamed := &RequestActor{PrincipalType: PrincipalTypeAgent, UserID: pulid.MustNew("usr_")}
	audited = unnamed.AuditActor()
	assert.Equal(t, AgentPrincipalID, audited.PrincipalID)
	assert.Equal(t, unnamed.UserID, audited.UserID)

	selfNamed := &RequestActor{PrincipalType: PrincipalTypeAgent}
	selfNamed.UserID = pulid.MustNew("usr_")
	selfNamed.PrincipalID = selfNamed.UserID
	assert.Equal(t, pulid.Nil, selfNamed.AuditActor().UserID,
		"an agent principal is never also the user it is recorded against")

	system := actorOf(PrincipalTypeSystem)
	assert.Equal(t, pulid.Nil, system.AuditActor().UserID)
}
