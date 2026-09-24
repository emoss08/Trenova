package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type delegateDefinitionsRepo struct {
	repositories.AgentDefinitionRepository

	byID  map[pulid.ID]*agentdefinition.Definition
	reads int
}

func (r *delegateDefinitionsRepo) ListByIDs(
	_ context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	r.reads++
	out := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := r.byID[id]; ok {
			out = append(out, definition)
		}
	}

	return out, nil
}

type delegatePermissions struct {
	services.PermissionEngine

	granted []pulid.ID
}

func (p *delegatePermissions) AgentsUsable(
	context.Context,
	*services.RequestActor,
	permission.Operation,
) (*services.UsableAgents, error) {
	return &services.UsableAgents{Assistant: true, GrantedIDs: p.granted}, nil
}

/*
MyAgent.delegates is read by everyone who may use the assistant, so it serves
only the delegates the reader may use: in the order configured, open to them
or granted to one of their roles, enabled and talked to. A delegate withheld
from them, turned off, moved to a schedule or deleted is simply absent, and so
is the agent itself should its own list name it.
*/
func TestMyAgentDelegates_ServesOnlyDelegatesTheReaderMayUse(t *testing.T) {
	t.Parallel()

	chat := func(name string, mode agentdefinition.AccessMode) *agentdefinition.Definition {
		return &agentdefinition.Definition{
			ID:          pulid.MustNew("agdef_"),
			Name:        name,
			Enabled:     true,
			TriggerMode: agentdefinition.TriggerChat,
			AccessMode:  mode,
		}
	}
	primary := chat("Homepage Widget Builder", agentdefinition.AccessEveryone)
	reports := chat("Report Builder", agentdefinition.AccessEveryone)
	payroll := chat("Payroll", agentdefinition.AccessRoles)
	treasury := chat("Treasury", agentdefinition.AccessRoles)
	off := chat("Old Desk", agentdefinition.AccessEveryone)
	off.Enabled = false
	nightly := chat("Nightly", agentdefinition.AccessEveryone)
	nightly.TriggerMode = agentdefinition.TriggerScheduled
	primary.DelegateIDs = []pulid.ID{
		payroll.ID, treasury.ID, off.ID, pulid.MustNew("agdef_"), nightly.ID, primary.ID, reports.ID,
	}

	repo := &delegateDefinitionsRepo{byID: map[pulid.ID]*agentdefinition.Definition{}}
	for _, definition := range []*agentdefinition.Definition{
		primary, reports, payroll, treasury, off, nightly,
	} {
		repo.byID[definition.ID] = definition
	}
	factory := loaders.NewUsableAgentByIDLoaderFactory(loaders.UsableAgentByIDLoaderFactoryParams{
		Definitions: repo,
		Permissions: &delegatePermissions{granted: []pulid.ID{payroll.ID}},
	})
	ctx := loaders.WithLoaders(t.Context(), &loaders.Loaders{
		UsableAgentByID: factory.NewForTenant(pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		}),
	})

	delegates, err := myAgentDelegates(ctx, primary)
	require.NoError(t, err)

	names := make([]string, 0, len(delegates))
	for _, delegate := range delegates {
		names = append(names, delegate.Name)
	}
	assert.Equal(t, []string{"Payroll", "Report Builder"}, names)
	assert.Equal(t, 1, repo.reads, "every delegate is read in one batch")
}

func TestMyAgentDelegates_NoAllowlistReadsNothing(t *testing.T) {
	t.Parallel()

	delegates, err := myAgentDelegates(t.Context(), &agentdefinition.Definition{})

	require.NoError(t, err)
	assert.Empty(t, delegates)
	assert.NotNil(t, delegates, "an empty list, never null, for a non-null field")
}
