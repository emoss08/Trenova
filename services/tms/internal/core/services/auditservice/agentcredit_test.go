package auditservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	req repositories.GetAgentDefinitionByIDRequest,
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

func unattendedParams(definitionID, systemUser pulid.ID) *services.LogActionParams {
	actor := (&services.RequestActor{
		PrincipalType:  services.PrincipalTypeAgent,
		PrincipalID:    definitionID,
		UserID:         systemUser,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}).AuditActor()

	return &services.LogActionParams{
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
) (*service, *mockBufferRepository) {
	t.Helper()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})
	srv.agents = newAgentCredits(definitions, zap.NewNop())

	return srv, bufferRepo
}

func pushedEntry(t *testing.T, bufferRepo *mockBufferRepository) *audit.Entry {
	t.Helper()

	var pushed *audit.Entry
	for _, call := range bufferRepo.Calls {
		if call.Method == "Push" {
			entry, ok := call.Arguments.Get(1).(*audit.Entry)
			require.True(t, ok)
			pushed = entry
		}
	}
	require.NotNil(t, pushed, "the entry was buffered")

	return pushed
}

func TestLogAction_AnUnattendedWriteNamesTheSystemUserAndCreditsTheAgent(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	systemUser := pulid.MustNew("usr_")
	definitions := &fakeAgentDefinitions{names: map[pulid.ID]string{definitionID: "Dispatch Agent"}}
	srv, bufferRepo := creditedService(t, definitions)
	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	params := unattendedParams(definitionID, systemUser)
	require.NoError(t, srv.LogAction(params, WithComment("Shipment updated")))

	entry := pushedEntry(t, bufferRepo)
	assert.Equal(t, string(services.PrincipalTypeAgent), entry.PrincipalType)
	assert.Equal(t, definitionID, entry.PrincipalID, "the principal names which agent ran")
	assert.Equal(t, systemUser, entry.UserID, "the user is the system account")
	assert.Equal(t, "Shipment updated (Ran by Dispatch Agent)", entry.Comment)
	require.Len(t, definitions.tenants, 1)
	assert.Equal(t, params.OrganizationID, definitions.tenants[0].OrganizationID,
		"the name is read in the entry's own tenant")
	assert.Equal(t, params.BusinessUnitID, definitions.tenants[0].BusinessUnitID)
}

func TestLogAction_AnAgentsNameIsReadOncePerDefinition(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	definitions := &fakeAgentDefinitions{names: map[pulid.ID]string{definitionID: "Billing Agent"}}
	srv, bufferRepo := creditedService(t, definitions)
	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	params := unattendedParams(definitionID, pulid.MustNew("usr_"))
	require.NoError(t, srv.LogAction(params))
	require.NoError(t, srv.LogAction(params))

	assert.Equal(t, 1, definitions.reads)
	assert.Equal(t, "Ran by Billing Agent", pushedEntry(t, bufferRepo).Comment)
}

func TestLogAction_AnAgentWhoseNameCannotBeReadIsStillCredited(t *testing.T) {
	t.Parallel()

	definitions := &fakeAgentDefinitions{err: errors.New("connection refused")}
	srv, bufferRepo := creditedService(t, definitions)
	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	params := unattendedParams(pulid.MustNew("agdef_"), pulid.MustNew("usr_"))
	require.NoError(t, srv.LogAction(params, WithComment("Hold released")))

	assert.Equal(t, "Hold released (Ran by an agent)", pushedEntry(t, bufferRepo).Comment)
}

func TestLogAction_AGenericAgentPrincipalIsCreditedWithoutALookup(t *testing.T) {
	t.Parallel()

	definitions := &fakeAgentDefinitions{}
	srv, bufferRepo := creditedService(t, definitions)
	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	params := unattendedParams(services.AgentPrincipalID, pulid.Nil)
	require.NoError(t, srv.LogAction(params))

	entry := pushedEntry(t, bufferRepo)
	assert.Equal(t, "Ran by an agent", entry.Comment)
	assert.Equal(t, pulid.Nil, entry.UserID)
	assert.Zero(t, definitions.reads)
}

func TestLogAction_APersonsWriteIsNotCredited(t *testing.T) {
	t.Parallel()

	definitions := &fakeAgentDefinitions{}
	srv, bufferRepo := creditedService(t, definitions)
	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	require.NoError(t, srv.LogAction(validLogActionParams(), WithComment("User created")))

	assert.Equal(t, "User created", pushedEntry(t, bufferRepo).Comment)
	assert.Zero(t, definitions.reads)
}

func TestLogActions_CreditsEachUnattendedEntry(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agdef_")
	definitions := &fakeAgentDefinitions{names: map[pulid.ID]string{definitionID: "Dispatch Agent"}}
	srv, bufferRepo := creditedService(t, definitions)

	var pushed []*audit.Entry
	bufferRepo.On("PushBatch", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entries, ok := args.Get(1).([]*audit.Entry)
			require.True(t, ok)
			pushed = entries
		}).
		Return(nil)

	require.NoError(t, srv.LogActions([]services.BulkLogEntry{
		{Params: unattendedParams(definitionID, pulid.MustNew("usr_"))},
		{Params: validLogActionParams()},
	}))

	require.Len(t, pushed, 2)
	assert.Equal(t, "Ran by Dispatch Agent", pushed[0].Comment)
	assert.Empty(t, pushed[1].Comment)
}
