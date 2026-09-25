package agentdefinitionservice

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type systemStore struct {
	repositories.AgentDefinitionRepository

	mu       sync.Mutex
	byKey    map[string]*agentdefinition.Definition
	names    map[string]struct{}
	inserted int
	reads    int
}

func newSystemStore(names ...string) *systemStore {
	store := &systemStore{
		byKey: map[string]*agentdefinition.Definition{},
		names: map[string]struct{}{},
	}
	for _, name := range names {
		store.names[strings.ToLower(name)] = struct{}{}
	}

	return store
}

func (s *systemStore) GetBySystemKey(
	_ context.Context,
	req repositories.GetAgentDefinitionBySystemKeyRequest,
) (*agentdefinition.Definition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reads++
	if found, ok := s.byKey[req.SystemKey]; ok {
		return found, nil
	}

	return nil, errortypes.NewNotFoundError("AgentDefinition not found within your organization")
}

func (s *systemStore) CreateSystem(
	_ context.Context,
	entity *agentdefinition.Definition,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, taken := s.byKey[entity.SystemKey]; taken {
		return false, nil
	}
	if _, taken := s.names[strings.ToLower(entity.Name)]; taken {
		return false, nil
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("agdef_")
	}
	s.byKey[entity.SystemKey] = entity
	s.names[strings.ToLower(entity.Name)] = struct{}{}
	s.inserted++

	return true, nil
}

type auditLog struct {
	services.AuditService

	mu      sync.Mutex
	entries []*services.LogActionParams
}

func (a *auditLog) LogAction(params *services.LogActionParams, _ ...services.LogOption) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, params)

	return nil
}

func systemTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestEnsureSystem_CreatesTheAgentFromItsTemplateOnce(t *testing.T) {
	t.Parallel()

	store := newSystemStore()
	audit := &auditLog{}
	s := &Service{l: zap.NewNop(), repo: store, audit: audit}
	tenant := systemTenant()

	first, err := s.EnsureSystem(t.Context(), services.EnsureSystemAgentRequest{
		TenantInfo: tenant,
		SystemKey:  agentdefinition.SystemKeyImportAssistant,
	})
	require.NoError(t, err)
	second, err := s.EnsureSystem(t.Context(), services.EnsureSystemAgentRequest{
		TenantInfo: tenant,
		SystemKey:  agentdefinition.SystemKeyImportAssistant,
	})
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, 1, store.inserted)
	require.Len(t, audit.entries, 1, "the first use is audited, later ones are reads")

	template := agentdefinition.TemplateImportAssistant
	assert.Equal(t, template.Label(), first.Name)
	assert.Equal(t, template, first.Template)
	assert.Equal(t, template.StarterTools(), first.ToolNames)
	assert.Equal(t, template.StarterInstructions(), first.Instructions)
	assert.Equal(t, template.StarterCeiling(), first.AutonomyCeiling)
	assert.Equal(t, agentdefinition.TriggerChat, first.TriggerMode)
	assert.Equal(t, agentdefinition.AccessEveryone, first.AccessMode)
	assert.True(t, first.Enabled)
	assert.False(t, first.ShadowMode)
	assert.True(t, first.IsPageAgent())
	assert.True(t, first.HasContextProvider(agentdefinition.ContextMemory), "memory is on")
	assert.True(t, first.HasContextProvider(agentdefinition.ContextPage))
	assert.Equal(t, tenant.OrgID, first.OrganizationID)
	assert.Equal(t, tenant.BuID, first.BusinessUnitID)
}

func TestEnsureSystem_ConcurrentFirstUsesShareOneAgent(t *testing.T) {
	t.Parallel()

	store := newSystemStore()
	s := &Service{l: zap.NewNop(), repo: store, audit: &auditLog{}}
	tenant := systemTenant()

	const callers = 16
	ids := make(chan pulid.ID, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			found, err := s.EnsureSystem(t.Context(), services.EnsureSystemAgentRequest{
				TenantInfo: tenant,
				SystemKey:  agentdefinition.SystemKeyFormulaAssistant,
			})
			if err == nil {
				ids <- found.ID
			}
		})
	}
	wg.Wait()
	close(ids)

	seen := map[pulid.ID]struct{}{}
	count := 0
	for id := range ids {
		seen[id] = struct{}{}
		count++
	}
	assert.Equal(t, callers, count)
	assert.Len(t, seen, 1)
	assert.Equal(t, 1, store.inserted)
}

func TestEnsureSystem_TakesAnotherNameWhenOneIsInUse(t *testing.T) {
	t.Parallel()

	label := agentdefinition.TemplateFormulaAssistant.Label()
	store := newSystemStore(label)
	s := &Service{l: zap.NewNop(), repo: store, audit: &auditLog{}}

	found, err := s.EnsureSystem(t.Context(), services.EnsureSystemAgentRequest{
		TenantInfo: systemTenant(),
		SystemKey:  agentdefinition.SystemKeyFormulaAssistant,
	})
	require.NoError(t, err)
	assert.Equal(t, label+" (built-in)", found.Name)

	crowded := newSystemStore(label, label+" (built-in)", label+" (built-in 2)",
		label+" (built-in 3)", label+" (built-in 4)")
	s.repo = crowded
	_, err = s.EnsureSystem(t.Context(), services.EnsureSystemAgentRequest{
		TenantInfo: systemTenant(),
		SystemKey:  agentdefinition.SystemKeyFormulaAssistant,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be set up")
}

func TestEnsureSystem_RefusesAnAgentItDoesNotCreate(t *testing.T) {
	t.Parallel()

	store := newSystemStore()
	s := &Service{l: zap.NewNop(), repo: store, audit: &auditLog{}}

	_, err := s.EnsureSystem(t.Context(), services.EnsureSystemAgentRequest{
		TenantInfo: systemTenant(),
		SystemKey:  "billing_exception",
	})
	require.Error(t, err)

	_, err = s.EnsureSystem(t.Context(), services.EnsureSystemAgentRequest{
		SystemKey: agentdefinition.SystemKeyImportAssistant,
	})
	require.Error(t, err)
	assert.Zero(t, store.inserted)
}
