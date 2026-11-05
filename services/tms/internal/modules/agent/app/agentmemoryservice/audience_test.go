package agentmemoryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type audienceRepo struct {
	agentmemoryports.AgentMemoryRepository

	memory *agentmemory.Memory
	listed *agentmemoryports.ListAgentMemoryConnectionRequest
}

func (r *audienceRepo) GetByID(
	context.Context,
	agentmemoryports.GetAgentMemoryByIDRequest,
) (*agentmemory.Memory, error) {
	return r.memory, nil
}

func (r *audienceRepo) ListConnection(
	_ context.Context,
	req *agentmemoryports.ListAgentMemoryConnectionRequest,
) (*pagination.CursorListResult[*agentmemory.Memory], error) {
	r.listed = req

	return &pagination.CursorListResult[*agentmemory.Memory]{}, nil
}

func audienceService(repo *audienceRepo, manages bool) *Service {
	return wired(&Service{
		l:           zap.NewNop(),
		repo:        repo,
		permissions: &fakePermissions{manages: manages},
	})
}

func audienceTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestGetByID_APersonalMemoryIsReadByWhoItIsKeptForOrWhoManagesMemory(t *testing.T) {
	t.Parallel()

	reader := userActor()
	someoneElse := pulid.MustNew("usr_")
	apiKey := &permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeAPIKey}

	tests := []struct {
		name    string
		memory  *agentmemory.Memory
		actor   *permissionports.RequestActor
		manages bool
		visible bool
	}{
		{
			name:    "the organization's memory reaches anyone who may read memories",
			memory:  &agentmemory.Memory{Scope: agentmemory.MemoryScopeOrganization},
			actor:   reader,
			visible: true,
		},
		{
			name:    "an agent's memory reaches anyone who may read memories",
			memory:  &agentmemory.Memory{Scope: agentmemory.MemoryScopeAgent},
			actor:   reader,
			visible: true,
		},
		{
			name: "the person's own memory",
			memory: &agentmemory.Memory{
				Scope:       agentmemory.MemoryScopeUser,
				OwnerUserID: &reader.UserID,
			},
			actor:   reader,
			visible: true,
		},
		{
			name: "someone else's memory",
			memory: &agentmemory.Memory{
				Scope:       agentmemory.MemoryScopeUser,
				OwnerUserID: &someoneElse,
			},
			actor: reader,
		},
		{
			name: "someone else's memory, read by a person who manages memory",
			memory: &agentmemory.Memory{
				Scope:       agentmemory.MemoryScopeUser,
				OwnerUserID: &someoneElse,
			},
			actor:   reader,
			manages: true,
			visible: true,
		},
		{
			name: "someone else's memory, read with an API key",
			memory: &agentmemory.Memory{
				Scope:       agentmemory.MemoryScopeUser,
				OwnerUserID: &someoneElse,
			},
			actor: apiKey,
		},
		{
			name: "a role the person does not hold",
			memory: &agentmemory.Memory{
				Scope:  agentmemory.MemoryScopeRole,
				RoleID: new(pulid.MustNew("rol_")),
			},
			actor: reader,
		},
		{
			name: "someone else's memory with no actor named",
			memory: &agentmemory.Memory{
				Scope:       agentmemory.MemoryScopeUser,
				OwnerUserID: &someoneElse,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := audienceService(&audienceRepo{memory: tt.memory}, tt.manages)

			got, err := svc.GetByID(t.Context(), agentmemoryports.GetAgentMemoryByIDRequest{
				ID:         pulid.MustNew("amem_"),
				TenantInfo: audienceTenant(),
				Actor:      tt.actor,
			})

			if tt.visible {
				require.NoError(t, err)
				assert.Same(t, tt.memory, got)
				return
			}
			require.Error(t, err)
			assert.True(t, errortypes.IsNotFoundError(err), "a hidden memory is not found")
			assert.Nil(t, got)
		})
	}
}

func TestListConnection_ReadsEveryAudienceOnlyForAPersonWhoManagesMemory(t *testing.T) {
	t.Parallel()

	t.Run("a reader sees what reaches them", func(t *testing.T) {
		t.Parallel()

		repo := &audienceRepo{}
		actor := userActor()

		_, err := audienceService(repo, false).ListConnection(
			t.Context(),
			&agentmemoryports.ListAgentMemoryConnectionRequest{
				TenantInfo: audienceTenant(),
				Actor:      actor,
			},
		)

		require.NoError(t, err)
		assert.False(t, repo.listed.AllAudiences)
		assert.Equal(t, actor.UserID, repo.listed.Reader.UserID)
	})

	t.Run("a request cannot ask for every audience itself", func(t *testing.T) {
		t.Parallel()

		repo := &audienceRepo{}

		_, err := audienceService(repo, false).ListConnection(
			t.Context(),
			&agentmemoryports.ListAgentMemoryConnectionRequest{
				TenantInfo:   audienceTenant(),
				Actor:        userActor(),
				AllAudiences: true,
			},
		)

		require.NoError(t, err)
		assert.False(t, repo.listed.AllAudiences)
	})

	t.Run("a person who manages memory sees them all", func(t *testing.T) {
		t.Parallel()

		repo := &audienceRepo{}

		_, err := audienceService(repo, true).ListConnection(
			t.Context(),
			&agentmemoryports.ListAgentMemoryConnectionRequest{
				TenantInfo: audienceTenant(),
				Actor:      userActor(),
			},
		)

		require.NoError(t, err)
		assert.True(t, repo.listed.AllAudiences)
	})

	t.Run("an API key reads as nobody in particular", func(t *testing.T) {
		t.Parallel()

		repo := &audienceRepo{}

		_, err := audienceService(repo, false).ListConnection(
			t.Context(),
			&agentmemoryports.ListAgentMemoryConnectionRequest{
				TenantInfo: audienceTenant(),
				Actor: &permissionports.RequestActor{
					PrincipalType: permissionports.PrincipalTypeAPIKey,
				},
			},
		)

		require.NoError(t, err)
		assert.False(t, repo.listed.AllAudiences)
		assert.True(t, repo.listed.Reader.UserID.IsNil())
	})
}
