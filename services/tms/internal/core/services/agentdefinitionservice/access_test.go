package agentdefinitionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingAccess struct {
	serviceports.AgentAccessService

	requests []*serviceports.SaveAgentWithAccessRequest
	refusal  error
}

func (a *recordingAccess) SaveWithAccess(
	ctx context.Context,
	req *serviceports.SaveAgentWithAccessRequest,
	_ *serviceports.RequestActor,
) (*agentdefinition.Definition, error) {
	a.requests = append(a.requests, req)
	if a.refusal != nil {
		return nil, a.refusal
	}

	return req.Save(ctx)
}

func savedAgent() func(context.Context) (*agentdefinition.Definition, error) {
	agent := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Payroll desk"}

	return func(context.Context) (*agentdefinition.Definition, error) {
		return agent, nil
	}
}

/*
A save that says who may use the agent goes through the access service, which
runs the write inside the transaction it sets the access in and holds the rule
for who may change it. A save that says nothing writes the agent alone.
*/
func TestSave_SetsAccessOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	access := &recordingAccess{}
	svc := &Service{l: zap.NewNop(), access: access}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	saved, err := svc.save(t.Context(), &serviceports.SaveAgentDefinitionRequest{
		TenantInfo: tenant,
	}, nil, savedAgent())
	require.NoError(t, err)
	assert.Equal(t, "Payroll desk", saved.Name)
	assert.Empty(t, access.requests, "a save that says nothing about access writes the agent alone")

	roles := []pulid.ID{pulid.MustNew("rol_")}
	saved, err = svc.save(t.Context(), &serviceports.SaveAgentDefinitionRequest{
		TenantInfo: tenant,
		Access: &serviceports.AgentAccessWrite{
			Mode:    agentdefinition.AccessRoles,
			RoleIDs: roles,
		},
	}, nil, savedAgent())
	require.NoError(t, err)
	assert.Equal(t, "Payroll desk", saved.Name)
	require.Len(t, access.requests, 1)
	assert.Equal(t, tenant, access.requests[0].TenantInfo)
	assert.Equal(t, agentdefinition.AccessRoles, access.requests[0].Access.Mode)
	assert.Equal(t, roles, access.requests[0].Access.RoleIDs)
}

// A refusal of the access is the save's refusal: the caller hears it, and
// nothing is scheduled or audited for an agent that was not saved.
func TestSave_AnAccessRefusalRefusesTheSave(t *testing.T) {
	t.Parallel()

	refusal := errortypes.NewAuthorizationError("needs role:update")
	svc := &Service{l: zap.NewNop(), access: &recordingAccess{refusal: refusal}}

	_, err := svc.save(t.Context(), &serviceports.SaveAgentDefinitionRequest{
		Access: &serviceports.AgentAccessWrite{Mode: agentdefinition.AccessRoles},
	}, nil, savedAgent())

	require.ErrorIs(t, err, refusal)
}

func TestSave_WithoutTheAccessServiceAccessIsRefused(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop()}

	_, err := svc.save(t.Context(), &serviceports.SaveAgentDefinitionRequest{
		Access: &serviceports.AgentAccessWrite{Mode: agentdefinition.AccessEveryone},
	}, nil, savedAgent())

	require.Error(t, err, "access asked for is never silently dropped")
}
