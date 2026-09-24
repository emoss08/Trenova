package agentdefinitionservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type sensitivityEngine struct {
	serviceports.PermissionEngine

	max permission.FieldSensitivity
	err error
}

func (e *sensitivityEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*serviceports.ResourcePermissionDetail, error) {
	if e.err != nil {
		return nil, e.err
	}

	return &serviceports.ResourcePermissionDetail{Resource: resource, MaxSensitivity: e.max}, nil
}

func personActor() *serviceports.RequestActor {
	user := pulid.MustNew("usr_")

	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    user,
		UserID:         user,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func withDataAccess(ceiling agentdefinition.DataAccessCeiling) *agentdefinition.Definition {
	return &agentdefinition.Definition{Name: "Payroll desk", DataAccessCeiling: ceiling}
}

func TestCheckDataAccess_OnlyAPersonWhoReachesRestrictedMayGrantIt(t *testing.T) {
	t.Parallel()

	internal := &Service{l: zap.NewNop(),
		permissions: &sensitivityEngine{max: permission.SensitivityInternal}}
	err := internal.checkDataAccess(t.Context(),
		withDataAccess(agentdefinition.DataAccessRestricted), nil, personActor())
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))
	assert.Equal(t, dataAccessField, multiErr.Errors[0].Field)

	restricted := &Service{l: zap.NewNop(),
		permissions: &sensitivityEngine{max: permission.SensitivityRestricted}}
	require.NoError(t, restricted.checkDataAccess(t.Context(),
		withDataAccess(agentdefinition.DataAccessRestricted), nil, personActor()))

	require.NoError(t, internal.checkDataAccess(t.Context(),
		withDataAccess(agentdefinition.DataAccessInternal), nil, personActor()),
		"lowering or keeping Internal needs nothing")
}

func TestCheckDataAccess_KeepingWhatTheAgentAlreadyHasIsNotAGrant(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(),
		permissions: &sensitivityEngine{max: permission.SensitivityInternal}}

	require.NoError(t, svc.checkDataAccess(t.Context(),
		withDataAccess(agentdefinition.DataAccessRestricted),
		withDataAccess(agentdefinition.DataAccessRestricted),
		personActor(),
	), "editing the name of a Restricted agent is not raising it")
}

func TestCheckDataAccess_RefusesWhenThePersonCannotBeRead(t *testing.T) {
	t.Parallel()

	failing := &Service{l: zap.NewNop(),
		permissions: &sensitivityEngine{err: errors.New("cache down")}}
	require.Error(t, failing.checkDataAccess(t.Context(),
		withDataAccess(agentdefinition.DataAccessRestricted), nil, personActor()))

	unwired := &Service{l: zap.NewNop()}
	require.Error(t, unwired.checkDataAccess(t.Context(),
		withDataAccess(agentdefinition.DataAccessRestricted), nil, personActor()),
		"with nothing to check against, nobody may raise an agent")
}

func TestApply_KeepsTheDataAccessWhenARequestLeavesItOut(t *testing.T) {
	t.Parallel()

	existing := withDataAccess(agentdefinition.DataAccessRestricted)
	apply(existing, &serviceports.SaveAgentDefinitionRequest{Name: "Payroll desk"})
	assert.Equal(t, agentdefinition.DataAccessRestricted, existing.DataAccessCeiling)

	created := &agentdefinition.Definition{}
	apply(created, &serviceports.SaveAgentDefinitionRequest{Name: "New desk"})
	assert.Equal(t, agentdefinition.DataAccessInternal, created.DataAccessCeiling)

	apply(existing, &serviceports.SaveAgentDefinitionRequest{
		Name:              "Payroll desk",
		DataAccessCeiling: agentdefinition.DataAccessInternal,
	})
	assert.Equal(t, agentdefinition.DataAccessInternal, existing.DataAccessCeiling)
}
