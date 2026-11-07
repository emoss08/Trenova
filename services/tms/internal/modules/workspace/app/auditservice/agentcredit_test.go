package auditservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeAgentDefinitions struct {
	mu      sync.Mutex
	names   map[pulid.ID]string
	err     error
	reads   int
	tenants []dbscope.Tenant
}

func (f *fakeAgentDefinitions) GetByID(
	ctx context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.reads++
	if tenant, ok := dbscope.TenantFrom(ctx); ok {
		f.tenants = append(f.tenants, tenant)
	}
	if f.err != nil {
		return nil, f.err
	}
	name, ok := f.names[req.ID]
	if !ok {
		return nil, errors.New("not found")
	}

	return &agentdefinition.Definition{ID: req.ID, Name: name}, nil
}

func unattendedParams(definitionID, systemUser pulid.ID) *auditports.LogActionParams {
	actor := (&permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    definitionID,
		UserID:         systemUser,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}).AuditActor()

	return &auditports.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     pulid.MustNew("shp_").String(),
		Operation:      permission.OpUpdate,
		CurrentState:   map[string]any{"status": "Assigned"},
		UserID:         actor.UserID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		APIKeyID:       actor.APIKeyID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func creditedService(
	t *testing.T,
	definitions *fakeAgentDefinitions,
) (*service, *fakeRepo) {
	t.Helper()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})
	srv.agents = newAgentCredits(definitions, zap.NewNop())

	return srv, repo
}

func pushedEntry(t *testing.T, repo *fakeRepo) *audit.Entry {
	t.Helper()

	entries := repo.entries()
	require.NotEmpty(t, entries, "the entry was stored")

	return entries[len(entries)-1]
}

func TestLogAction_AnUnattendedWriteNamesTheSystemUserAndCreditsTheAgent(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	systemUser := pulid.MustNew("usr_")
	definitions := &fakeAgentDefinitions{names: map[pulid.ID]string{definitionID: "Dispatch Agent"}}
	srv, repo := creditedService(t, definitions)

	params := unattendedParams(definitionID, systemUser)
	require.NoError(t, srv.LogAction(t.Context(), params, WithComment("Shipment updated")))

	entry := pushedEntry(t, repo)
	assert.Equal(t, string(permissionports.PrincipalTypeAgent), entry.PrincipalType)
	assert.Equal(t, definitionID, entry.PrincipalID, "the principal names which agent ran")
	assert.Equal(t, systemUser, entry.UserID, "the user is the system account")
	assert.Equal(t, "Shipment updated (Ran by Dispatch Agent)", entry.Comment)
	require.Len(t, definitions.tenants, 1)
	assert.Equal(t, params.OrganizationID, definitions.tenants[0].OrganizationID,
		"the name is read in the entry's own tenant")
	assert.Equal(t, params.BusinessUnitID, definitions.tenants[0].BusinessUnitID)
}

func TestLogAction_TheAgentsNameIsReadOnceAndCached(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	definitions := &fakeAgentDefinitions{names: map[pulid.ID]string{definitionID: "Billing Agent"}}
	srv, repo := creditedService(t, definitions)

	params := unattendedParams(definitionID, pulid.MustNew("usr_"))
	require.NoError(t, srv.LogAction(t.Context(), params))
	require.NoError(t, srv.LogAction(t.Context(), params))

	assert.Equal(t, 1, definitions.reads, "repeats are served from the in-process cache")
	assert.Equal(t, "Ran by Billing Agent", pushedEntry(t, repo).Comment)
}

func TestLogAction_AnAgentWhoseNameCannotBeReadIsStillCredited(t *testing.T) {
	t.Parallel()

	definitions := &fakeAgentDefinitions{err: errors.New("connection refused")}
	srv, repo := creditedService(t, definitions)

	params := unattendedParams(pulid.MustNew("agdef_"), pulid.MustNew("usr_"))
	require.NoError(t, srv.LogAction(t.Context(), params, WithComment("Hold released")))

	assert.Equal(t, "Hold released (Ran by an agent)", pushedEntry(t, repo).Comment)
}

func TestLogAction_AGenericAgentPrincipalIsCreditedWithoutALookup(t *testing.T) {
	t.Parallel()

	definitions := &fakeAgentDefinitions{}
	srv, repo := creditedService(t, definitions)

	params := unattendedParams(permissionports.AgentPrincipalID, pulid.Nil)
	require.NoError(t, srv.LogAction(t.Context(), params))

	entry := pushedEntry(t, repo)
	assert.Equal(t, "Ran by an agent", entry.Comment)
	assert.Equal(t, pulid.Nil, entry.UserID)
	assert.Zero(t, definitions.reads)
}

func TestLogAction_APersonsWriteIsNotCredited(t *testing.T) {
	t.Parallel()

	definitions := &fakeAgentDefinitions{}
	srv, repo := creditedService(t, definitions)

	require.NoError(t, srv.LogAction(t.Context(), validLogActionParams(), WithComment("User created")))

	assert.Equal(t, "User created", pushedEntry(t, repo).Comment)
	assert.Zero(t, definitions.reads)
}

func TestLogActions_CreditsEachUnattendedEntry(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	definitions := &fakeAgentDefinitions{names: map[pulid.ID]string{definitionID: "Dispatch Agent"}}
	srv, repo := creditedService(t, definitions)

	require.NoError(t, srv.LogActions(t.Context(), []auditports.BulkLogEntry{
		{Params: unattendedParams(definitionID, pulid.MustNew("usr_"))},
		{Params: validLogActionParams()},
	}))

	pushed := repo.entries()
	require.Len(t, pushed, 2)
	assert.Equal(t, "Ran by Dispatch Agent", pushed[0].Comment)
	assert.Empty(t, pushed[1].Comment)
}
